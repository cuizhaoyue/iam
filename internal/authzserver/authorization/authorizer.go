// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package authorization

import (
	authzv1 "github.com/marmotedu/api/authz/v1"
	"github.com/ory/ladon"

	"github.com/marmotedu/iam/pkg/log"
)

// Authorizer implement the authorize interface that use local repository to
// authorize the subject access review.
// Authorizer 实现了authorize接口，授权人
type Authorizer struct {
	warden ladon.Warden
}

// NewAuthorizer creates a local repository authorizer and returns it.
// 创建一个本地仓库授权人,包含 Manager 和 AuditLogger 字段
func NewAuthorizer(authorizationClient AuthorizationInterface) *Authorizer {
	return &Authorizer{
		warden: &ladon.Ladon{
			Manager:     NewPolicyManager(authorizationClient),
			AuditLogger: NewAuditLogger(authorizationClient),
		},
	}
}

// Authorize to determine the subject access.
// 执行授权操作，确认主题的访问权限
func (a *Authorizer) Authorize(request *ladon.Request) *authzv1.Response {
	log.Debug("authorize request", log.Any("request", request))

	// IsAllow会先调用Manager的FindRequestCandidates查询用户的策略列表
	// IsAllowed 会调用 DoPoliciesAllow(r, policies) 函数进行权限校验。
	// 如果权限校验不通过（请求在指定条件下不能够对资源做指定操作），就调用 LogRejectedAccessRequest
	// 函数记录拒绝的请求，并返回值为非 nil 的 error，error 中记录了授权失败的错误信息。如果权限校验通过，
	// 则调用 LogGrantedAccessRequest 函数记录允许的请求，并返回值为 nil 的 error。
	if err := a.warden.IsAllowed(request); err != nil {
		return &authzv1.Response{
			Denied: true,
			Reason: err.Error(),
		}
	}

	return &authzv1.Response{
		Allowed: true,
	}
}
