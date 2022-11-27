// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package authorization

import (
	"github.com/ory/ladon"

	"github.com/marmotedu/iam/pkg/log"
)

// AuditLogger outputs and cache information about granting or rejecting policies.
type AuditLogger struct {
	client AuthorizationInterface
}

// NewAuditLogger creates a AuditLogger with default parameters.
// 创建一个带有默认参数的审计Logger实例
func NewAuditLogger(client AuthorizationInterface) *AuditLogger {
	return &AuditLogger{
		client: client,
	}
}

// LogRejectedAccessRequest write rejected subject access to log.
// 记录被拒绝的访问请求
func (a *AuditLogger) LogRejectedAccessRequest(r *ladon.Request, p ladon.Policies, d ladon.Policies) {
	a.client.LogRejectedAccessRequest(r, p, d)
	log.Debug("subject access review rejected", log.Any("request", r), log.Any("deciders", d))
}

// LogGrantedAccessRequest write granted subject access to log.
// 记录允许的访问请求
func (a *AuditLogger) LogGrantedAccessRequest(r *ladon.Request, p ladon.Policies, d ladon.Policies) {
	a.client.LogGrantedAccessRequest(r, p, d)
	log.Debug("subject access review granted", log.Any("request", r), log.Any("deciders", d))
}
