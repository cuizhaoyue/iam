// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package user

import (
	srvv1 "github.com/marmotedu/iam/internal/apiserver/service/v1"
	"github.com/marmotedu/iam/internal/apiserver/store"
)

// UserController create a user handler used to handle request for user resource.
// 创建一个user处理器用于处理user资源相关的请求，控制器成员是服务接口实例
type UserController struct {
	srv srvv1.Service
}

// NewUserController creates a user handler.
// 创建一个用户处理器，传入参数是仓库层的mysql工厂类型，创建控制器时成员实例需要mysql工厂实例作为参数
func NewUserController(store store.Factory) *UserController {
	return &UserController{
		srv: srvv1.NewService(store),
	}
}
