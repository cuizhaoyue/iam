// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package v1

//go:generate mockgen -self_package=github.com/marmotedu/iam/internal/apiserver/service/v1 -destination mock_service.go -package v1 github.com/marmotedu/iam/internal/apiserver/service/v1 Service,UserSrv,SecretSrv,PolicySrv

import "github.com/marmotedu/iam/internal/apiserver/store"

// Service defines functions used to return resource interface.
// 定义返回资源接口的功能
type Service interface {
	Users() UserSrv
	Secrets() SecretSrv
	Policies() PolicySrv
}

// 整体服务实例
type service struct {
	store store.Factory
}

// NewService returns Service interface. 创建服务实例
func NewService(store store.Factory) Service {
	return &service{
		store: store,
	}
}

// Users 创建User相关的服务
func (s *service) Users() UserSrv {
	return newUsers(s)
}

// Secrets 创建secret相关的服务实例
func (s *service) Secrets() SecretSrv {
	return newSecrets(s)
}

// Policies 创建policy相关的服务实例
func (s *service) Policies() PolicySrv {
	return newPolicies(s)
}
