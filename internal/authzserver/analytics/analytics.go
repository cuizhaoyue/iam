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
	store                      storage.AnalyticsHandler // analytics处理器
	poolSize                   int                      // 线程池大小，表示worker个数
	recordsChan                chan *AnalyticsRecord    // 记录数据的通道
	workerBufferSize           uint64                   // 数据缓冲区大小
	recordsBufferFlushInterval uint64                   // 数据记录时间间隔
	shouldStop                 uint32                   // 是否应该停止发送数据，大于0时停止
	poolWg                     sync.WaitGroup           // 管理线程池的WaitGroup
}

// NewAnalytics returns a new analytics instance.
// 创建一个Analytics实例
func NewAnalytics(options *AnalyticsOptions, store storage.AnalyticsHandler) *Analytics {
	ps := options.PoolSize
	recordsBufferSize := options.RecordsBufferSize
	workerBufferSize := recordsBufferSize / uint64(ps) // 每个worker可以缓存的日志消息数
	log.Debug("Analytics pool worker buffer size", log.Uint64("workerBufferSize", workerBufferSize))

	// 通过RecordHit函数，向recordsChan 中写入 AnalyticsRecord 类型的数据
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

// Start start the analytics service.
func (r *Analytics) Start() {
	r.store.Connect()

	// start worker pool
	// 启动工作池
	atomic.SwapUint32(&r.shouldStop, 0) // 保存新的值，返回旧的值
	for i := 0; i < r.poolSize; i++ {
		r.poolWg.Add(1)
		go r.recordWorker()
	}
}

// Stop stop the analytics service.
func (r *Analytics) Stop() {
	// flag to stop sending records into channel
	atomic.SwapUint32(&r.shouldStop, 1)

	// close channel to stop workers
	close(r.recordsChan)

	// wait for all workers to be done
	r.poolWg.Wait()
}

// RecordHit will store an AnalyticsRecord in Redis.
// 将授权日志转换成AnalyticsRecord对象保存到Redis中.
func (r *Analytics) RecordHit(record *AnalyticsRecord) error {
	// check if we should stop sending records 1st
	// 首先检查我们是否应该停止发送记录
	if atomic.LoadUint32(&r.shouldStop) > 0 {
		return nil
	}

	// just send record to channel consumed by pool of workers
	// leave all data crunching and Redis I/O work for pool workers
	// 发送记录到通道到工作线程池消费的通道
	r.recordsChan <- record

	return nil
}

// 用于记录数据的线程
func (r *Analytics) recordWorker() {
	defer r.poolWg.Done() // 退出时goroutine计数减1

	// this is buffer to send one pipelined command to redis
	// use r.recordsBufferSize as cap to reduce slice re-allocations
	recordsBuffer := make([][]byte, 0, r.workerBufferSize) // 设置缓存大小

	// read records from channel and process
	// 从通道和程序中读取记录数据.
	lastSentTS := time.Now() // 最后发送数据的时间
	for {
		var readyToSend bool // 是否准备好发送数据
		select {
		case record, ok := <-r.recordsChan:
			// check if channel was closed and it is time to exit from worker
			// 检查通道是否关闭，如果关闭则退出worker线程
			if !ok {
				// send what is left in buffer
				// 发送buffer中留下的数据后退出
				r.store.AppendToSetPipelined(analyticsKeyName, recordsBuffer)

				return
			}

			// we have new record - prepare it and add to buffer

			// 通道中已经有记录数据了就添加到buffer中
			if encoded, err := msgpack.Marshal(record); err != nil {
				log.Errorf("Error encoding analytics data: %s", err.Error())
			} else {
				recordsBuffer = append(recordsBuffer, encoded)
			}

			// identify that buffer is ready to be sent
			readyToSend = uint64(len(recordsBuffer)) == r.workerBufferSize

		case <-time.After(time.Duration(r.recordsBufferFlushInterval) * time.Millisecond):
			// 间隔时间到了以后就可以发送buffer中的数据了.

			// nothing was received for that period of time
			// anyways send whatever we have, don't hold data too long in buffer
			readyToSend = true
		}

		// send data to Redis and reset buffer
		// 发送数据到redis并且重置buffer
		if len(recordsBuffer) > 0 && (readyToSend || time.Since(lastSentTS) >= recordsBufferForcedFlushInterval) {
			r.store.AppendToSetPipelined(analyticsKeyName, recordsBuffer) // 发送数据到redis
			recordsBuffer = recordsBuffer[:0]                             // 清空buffer
			lastSentTS = time.Now()                                       // 重置时间
		}
	}
}

// DurationToMillisecond convert time duration type to float64.
func DurationToMillisecond(d time.Duration) float64 {
	return float64(d) / 1e6
}
