// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package auth

import (
	ginjwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"

	"github.com/marmotedu/iam/internal/pkg/middleware"
)

// AuthzAudience defines the value of jwt audience field. 定义jwt token接收者
const AuthzAudience = "iam.authz.marmotedu.com"

// JWTStrategy defines jwt bearer authentication strategy. 定义了jwt认证策略
type JWTStrategy struct {
	ginjwt.GinJWTMiddleware
}

var _ middleware.AuthStrategy = &JWTStrategy{}

// NewJWTStrategy create jwt bearer strategy with GinJWTMiddleware.
// 使用GinJWTMiddleware创建jwt认证策略
func NewJWTStrategy(gjwt ginjwt.GinJWTMiddleware) JWTStrategy {
	return JWTStrategy{gjwt}
}

// AuthFunc defines jwt bearer strategy as the gin authentication middleware.
// 定义jwt策略作为gin的认证中间件
func (j JWTStrategy) AuthFunc() gin.HandlerFunc {
	return j.MiddlewareFunc()
}
