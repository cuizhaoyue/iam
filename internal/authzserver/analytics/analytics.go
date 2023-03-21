// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package analytics defines functions and structs used to store authorization audit data to redis.
package analytics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/marmotedu/iam/pkg/log"
	"github.com/marmotedu/iam/pkg/storage"
)

const analyticsKeyName = "iam-system-analytics"

const (
	recordsBufferForcedFlushInterval = 1 * time.Second
)

// AnalyticsRecord encodes the details of a authorization request.
// AnalyticsRecord 编码授权请求的详细信息.
type AnalyticsRecord struct {
	TimeStamp  int64     `json:"timestamp"`                  // 时间戳
	Username   string    `json:"username"`                   // 授权请求中的用户名
	Effect     string    `json:"effect"`                     // 允许或拒绝 allow or deny
	Conclusion string    `json:"conclusion"`                 // 结论
	Request    string    `json:"request"`                    // 请求
	Policies   string    `json:"policies"`                   // 策略
	Deciders   string    `json:"deciders"`                   //
	ExpireAt   time.Time `json:"expireAt"   bson:"expireAt"` // 到期时间
}

// 全局变量，分析服务的配置
var analytics *Analytics

// SetExpiry set expiration time to a key.
// 设置到期时间
func (a *AnalyticsRecord) SetExpiry(expiresInSeconds int64) {
	expiry := time.Duration(expiresInSeconds) * time.Second
	if expiresInSeconds == 0 { // 如果传入0则设置100年过期时间
		// Expiry is set to 100 years
		expiry = 24 * 365 * 100 * time.Hour
	}

	t := time.Now()
	t2 := t.Add(expiry)
	a.ExpireAt = t2
}

// Analytics will record analytics data to a redis back end as defined in the Config object.
// 把分析数据按照Config对象中的定义记录到redis后端
type Analytics struct {
	store                      storage.AnalyticsHandler // storage.AnalyticsHandler接口实例，提供连接和投递给storage的函数
	poolSize                   int                      // 指定开启的worker数，也就是说开启多少个goroutine来消费recordsChan中的消息
	recordsChan                chan *AnalyticsRecord    // 记录数据的通道
	workerBufferSize           uint64                   // 批量投递给下游系统的消息数，通过批量投递可以进一步提高消费能力，减少cpu消耗
	recordsBufferFlushInterval uint64                   // 最迟多久投递一次，投递数据的超时时间，不能单纯的理解为时间间隔，因为还存在影响投递的其它条件
	shouldStop                 uint32
	poolWg                     sync.WaitGroup
}

// NewAnalytics returns a new analytics instance.
// 创建一个Analytics实例
func NewAnalytics(options *AnalyticsOptions, store storage.AnalyticsHandler) *Analytics {
	ps := options.PoolSize
	recordsBufferSize := options.RecordsBufferSize
	workerBufferSize := recordsBufferSize / uint64(ps) // 每个worker可以缓存的日志消息数
	log.Debug("Analytics pool worker buffer size", log.Uint64("workerBufferSize", workerBufferSize))

	// 授权日志缓存在recordsChan中，其长度通过配置文件设置
	recordsChan := make(chan *AnalyticsRecord, recordsBufferSize)

	analytics = &Analytics{
		store:                      store,
		poolSize:                   ps,
		recordsChan:                recordsChan,
		workerBufferSize:           workerBufferSize,
		recordsBufferFlushInterval: options.FlushInterval,
	}

	return analytics
}

// GetAnalytics returns the existed analytics instance.
// Need to initialize `analytics` instance before calling GetAnalytics.
func GetAnalytics() *Analytics {
	return analytics
}

// Start 启动数据上报服务
func (r *Analytics) Start() {
	r.store.Connect()

	// 启动工作池
	atomic.SwapUint32(&r.shouldStop, 0) // 设置允许向recordsChan中添加数据的标志位
	for i := 0; i < r.poolSize; i++ {
		r.poolWg.Add(1)
		go r.recordWorker() // 启动多个协和共同消费recordsChan中的消息
	}
}

// Stop 停止数据上报服务
// 主程序收到系统终止命令后，调用Stop优雅关停数据上报服务，确定缓存中的数据都能上报成功
func (r *Analytics) Stop() {
	// 设置停止给channel发送信息的标志，1-停止发送
	atomic.SwapUint32(&r.shouldStop, 1)

	// close channel to stop workers
	// 关闭channel来停止worker工作
	close(r.recordsChan)

	// wait for all workers to be done
	r.poolWg.Wait()
}

// RecordHit 记录AnalyticsRecord 的数据
func (r *Analytics) RecordHit(record *AnalyticsRecord) error {
	// check if we should stop sending records 1st
	// 首先检查此时状态是否在关停中，如果在关停中则丢弃数据，1-代表服务关停
	if atomic.LoadUint32(&r.shouldStop) > 0 {
		return nil
	}

	// just send record to channel consumed by pool of workers
	// leave all data crunching and Redis I/O work for pool workers
	// 发送记录到通道到工作线程池消费的通道
	r.recordsChan <- record

	return nil
}

// 消费逻辑，消费recordsChan中的消息
func (r *Analytics) recordWorker() {
	defer r.poolWg.Done() // 退出时goroutine计数减1

	// 这是向 Redis 发送一个流水线命令的缓冲区
	// 使用 r.recordsBufferSize 作为容量以减少切片的重新分配
	recordsBuffer := make([][]byte, 0, r.workerBufferSize)

	// read records from channel and process
	// 从通道和程序中读取记录数据.
	lastSentTS := time.Now() // 最后发送数据的时间
	for {
		var readyToSend bool // 是否准备好发送数据
		select {
		case record, ok := <-r.recordsChan:
			// 检查通道是否关闭，如果关闭则退出worker线程
			if !ok {
				// channel关闭后把剩余的消息上报给storage，然后退出
				r.store.AppendToSetPipelined(analyticsKeyName, recordsBuffer)

				return
			}

			// 有新的消息后-准备把它添加到buffer中

			if encoded, err := msgpack.Marshal(record); err != nil {
				log.Errorf("Error encoding analytics data: %s", err.Error())
			} else {
				recordsBuffer = append(recordsBuffer, encoded)
			}

			// 校验是否可以发送buffer中的消息，buffer中的消息数到达最大worker可处理的消息长度即可投递
			readyToSend = uint64(len(recordsBuffer)) == r.workerBufferSize

		case <-time.After(time.Duration(r.recordsBufferFlushInterval) * time.Millisecond):
			// 到达消息投递的超时时间也可认为可以发送buffer中的消息了

			// nothing was received for that period of time
			// anyways send whatever we have, don't hold data too long in buffer
			readyToSend = true
		}

		// send data to Redis and reset buffer
		// 发送数据到redis并且重置buffer，如果投递超时时间超过1s则每次投递一次
		// recordsBufferForcedFlushInterval表示最大的投递超时时间，防止配置文件将 recordsBufferFlushInterval 设得过大。
		if len(recordsBuffer) > 0 && (readyToSend || time.Since(lastSentTS) >= recordsBufferForcedFlushInterval) {
			r.store.AppendToSetPipelined(analyticsKeyName, recordsBuffer) // 发送数据到redis
			recordsBuffer = recordsBuffer[:0]                             // 清空buffer
			lastSentTS = time.Now()                                       // 重置时间
		}
	}
}

// DurationToMillisecond convert time duration type to float64.
// 没有用到，暂时注释
// func DurationToMillisecond(d time.Duration) float64 {
// 	return float64(d) / 1e6
// }
