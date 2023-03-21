// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package server

import (
	"os"
	"os/signal"
)

var onlyOneSignalHandler = make(chan struct{})

var shutdownHandler chan os.Signal

// SetupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
// 注册了SIGTERM 和 SIGINT两个信号，该函数返回一个stop
func SetupSignalHandler() <-chan struct{} {
	// 同一个channel只能被调用一次，再次执行close操作会panic，利用这个channel特性确保这个函数只被调用一次
	close(onlyOneSignalHandler) // panics when called twice

	shutdownHandler = make(chan os.Signal, 2) // 接收信号

	stop := make(chan struct{}) // 用于程序关闭的通道

	// signal.Notify(c chan<- os.Signal, sig ...os.Signal)函数不会为了向c中发送信息而阻塞
	// 也就是说如果发送时c阻塞了，signal包会直接丢弃信息，为了不丢失信息需要使用有缓存的channel
	signal.Notify(shutdownHandler, shutdownSignals...)

	go func() {
		// 接收一次信号程序关闭，接收两次信息程序退出
		<-shutdownHandler
		close(stop)
		<-shutdownHandler
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

// RequestShutdown emulates a received event that is considered as shutdown signal (SIGTERM/SIGINT)
// This returns whether a handler was notified.
// 没有用到这个函数，先注释掉
// func RequestShutdown() bool {
// 	if shutdownHandler != nil {
// 		select {
// 		case shutdownHandler <- shutdownSignals[0]:
// 			return true
// 		default:
// 		}
// 	}
//
// 	return false
// }
