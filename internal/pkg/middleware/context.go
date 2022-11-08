// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/marmotedu/iam/pkg/log"
)

// UsernameKey defines the key in gin context which represents the owner of the secret.
const UsernameKey = "username"

// Context is a middleware that injects common prefix fields to gin.Context.
// Context是一个中间件，它插入通用的前缀字段到gin.Context
func Context() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(log.KeyRequestID, c.GetString(XRequestIDKey)) // 上下文中设置requestID值为请求id
		c.Set(log.KeyUsername, c.GetString(UsernameKey))    // 设置username的值
		c.Next()
	}
}
