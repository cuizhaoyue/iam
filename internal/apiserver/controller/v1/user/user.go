// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package user

import (
	srvv1 "github.com/marmotedu/iam/internal/apiserver/service/v1"
	"github.com/marmotedu/iam/internal/apiserver/store"
)

// UserController create a user handler used to handle request for user resource.
// 创建一个user处理器用于处理user资源相关的请求，调用服务层
type UserController struct {
	srv srvv1.Service
}

// NewUserController creates a user handler. 创建一个用户处理器
func NewUserController(store store.Factory) *UserController {
	return &UserController{
		srv: srvv1.NewService(store),
	}
}
