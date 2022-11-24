// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package load

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/marmotedu/iam/pkg/log"
	"github.com/marmotedu/iam/pkg/storage"
)

// Loader defines function to reload storage.
type Loader interface {
	Reload() error
}

// Load is used to reload given storage.
// 用于重新加载存储数据
type Load struct {
	ctx    context.Context
	lock   *sync.RWMutex
	loader Loader
}

// NewLoader return a loader with a loader implement.
// 返回一个带有loader实现实例的加载器
func NewLoader(ctx context.Context, loader Loader) *Load {
	return &Load{
		ctx:    ctx,
		lock:   new(sync.RWMutex),
		loader: loader,
	}
}

// Start start a loop service.
func (l *Load) Start() {
	go startPubSubLoop()   // 订阅redis通道，注册回调函数判断是否需要同步密钥和策略
	go l.reloadQueueLoop() // 有新消息后，把新消息添加到requeue中
	// 1s is the minimum amount of time between hot reloads. The
	// interval counts from the start of one reload to the next.
	go l.reloadLoop() // 每隔1秒检查一次requeue是否为空，不为空则重新加载密钥和策略
	l.DoReload()      // 完成一次密钥和策略的同步
}

// 协程，订阅redis通道，注册回调函数，判断是否需要同步密钥和策略
func startPubSubLoop() {
	cacheStore := storage.RedisCluster{}
	cacheStore.Connect()
	// On message, synchronize
	for {
		// 订阅redis的channel并且注册一个回调函数，转换接收的消息判断是否需要同步密钥和策略
		err := cacheStore.StartPubSubHandler(RedisPubSubChannel, func(v interface{}) {
			handleRedisEvent(v, nil, nil)
		})
		if err != nil {
			if !errors.Is(err, storage.ErrRedisIsDown) {
				log.Errorf("Connection to Redis failed, reconnect in 10s: %s", err.Error())
			}

			time.Sleep(10 * time.Second)
			log.Warnf("Reconnecting: %s", err.Error())
		}
	}
}

// shouldReload returns true if we should perform any reload. Reloads happens if
// we have reload callback queued.
// requeue不为空时返回true
func shouldReload() ([]func(), bool) {
	requeueLock.Lock()
	defer requeueLock.Unlock()
	if len(requeue) == 0 {
		return nil, false
	}
	n := requeue
	requeue = []func(){} // requeue重新置空

	return n, true
}

func (l *Load) reloadLoop(complete ...func()) {
	ticker := time.NewTicker(1 * time.Second) // 定时器，1s间隔
	for {
		select {
		case <-l.ctx.Done():
			return
		// We don't check for reload right away as the gateway peroms this on the
		// startup sequence. We expect to start checking on the first tick after the
		// gateway is up and running.
		case <-ticker.C:
			cb, ok := shouldReload() // 检查requeue是否为空
			if !ok {
				continue
			}
			start := time.Now()
			l.DoReload() // requeue不为空时同步密钥和策略
			for _, c := range cb {
				// most of the callbacks are nil, we don't want to execute nil functions to
				// avoid panics.
				// 回调函数都是nil，不执行nil避免出现panic
				if c != nil {
					c()
				}
			}
			if len(complete) != 0 {
				complete[0]()
			}
			log.Infof("reload: cycle completed in %v", time.Since(start))
		}
	}
}

// reloadQueue used to queue a reload. It's not
// buffered, as reloadQueueLoop should pick these up immediately.
// reloadQueue 主要用来告诉程序，需要完成一次密钥和策略的同步。
var reloadQueue = make(chan func())

var requeueLock sync.Mutex

// This is a list of callbacks to execute on the next reload. It is protected by
// requeueLock for concurrent use.
// 这是一个回调函数的列表，在下次加载时将会被执行.它在被并发使用时是由requeueLock保护的.
var requeue []func()

// 协程，监听reloadQueue，当发现通道中有新消息(这里是空的回调函数)写入时，会实时将消息缓存到requeue切片中.
func (l *Load) reloadQueueLoop(cb ...func()) {
	for {
		select {
		case <-l.ctx.Done():
			return
		case fn := <-reloadQueue: // 有新消息出现，把消息添加到requeue中
			requeueLock.Lock() // append之前要上锁
			requeue = append(requeue, fn)
			requeueLock.Unlock()
			log.Info("Reload queued")
			if len(cb) != 0 {
				cb[0]()
			}
		}
	}
}

// DoReload reload secrets and policies.
func (l *Load) DoReload() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if err := l.loader.Reload(); err != nil {
		log.Errorf("faild to refresh target storage: %s", err.Error())
	}

	log.Debug("refresh target storage succ")
}
