// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package store

//go:generate mockgen -self_package=github.com/marmotedu/iam/internal/apiserver/store -destination mock_store.go -package store github.com/marmotedu/iam/internal/apiserver/store Factory,UserStore,SecretStore,PolicyStore

// 全局变量，工厂类型，操作所有的存储对象
var client Factory

// Factory defines the iam platform storage interface.
// 定义iam平台的存储接口，工厂模式，工厂中定义了操作所有对象的方法
type Factory interface {
	Users() UserStore
	Secrets() SecretStore
	Policies() PolicyStore
	PolicyAudits() PolicyAuditStore
	Close() error
}

// Client return the store client instance. 返回工厂实例
func Client() Factory {
	return client
}

// SetClient set the iam store client. 设置工厂实例
func SetClient(factory Factory) {
	client = factory
}
