// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package authorize implements the authorize handlers.
package authorize

import (
	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/core"
	"github.com/marmotedu/errors"
	"github.com/ory/ladon"

	"github.com/marmotedu/iam/internal/authzserver/authorization"
	"github.com/marmotedu/iam/internal/authzserver/authorization/authorizer"
	"github.com/marmotedu/iam/internal/pkg/code"
)

// AuthzController create a authorize handler used to handle authorize request.
// 创建一个授权处理handler处理授权请求
type AuthzController struct {
	store authorizer.PolicyGetter // authorizer属于服务层
}

// NewAuthzController creates a authorize handler.
// 创建一个授权处理器
func NewAuthzController(store authorizer.PolicyGetter) *AuthzController {
	return &AuthzController{
		store: store,
	}
}

// Authorize returns whether a request is allow or deny to access a resource and do some action
// under specified condition.
// 拒绝或允许某种条件下的请求访问某种资源或作其它操作.
func (a *AuthzController) Authorize(c *gin.Context) {
	var r ladon.Request
	if err := c.ShouldBind(&r); err != nil {
		core.WriteResponse(c, errors.WithCode(code.ErrBind, err.Error()), nil)

		return
	}

	// 创建并返回包含 Manager 和 AuditLogger 字段的Authorizer类型的变量。
	auth := authorization.NewAuthorizer(authorizer.NewAuthorization(a.store))
	if r.Context == nil {
		r.Context = ladon.Context{}
	}

	r.Context["username"] = c.GetString("username") // 从上下文中获取username
	rsp := auth.Authorize(&r)                       // 返回授权结果

	core.WriteResponse(c, nil, rsp)
}
