// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package secret

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/marmotedu/api/apiserver/v1"
	"github.com/marmotedu/component-base/pkg/core"
	metav1 "github.com/marmotedu/component-base/pkg/meta/v1"
	"github.com/marmotedu/errors"

	"github.com/marmotedu/iam/internal/pkg/code"
	"github.com/marmotedu/iam/internal/pkg/middleware"
	"github.com/marmotedu/iam/pkg/log"
)

// Update update a key by the secret key identifier.
func (s *SecretController) Update(c *gin.Context) {
	log.L(c).Info("update secret function called.")

	var r v1.Secret
	if err := c.ShouldBindJSON(&r); err != nil {
		core.WriteResponse(c, errors.WithCode(code.ErrBind, err.Error()), nil)

		return
	}

	username := c.GetString(middleware.UsernameKey)
	name := c.Param("name") // 从path参数中获取secret名称

	// 获取指定的secret
	secret, err := s.srv.Secrets().Get(c, username, name, metav1.GetOptions{})
	if err != nil {
		core.WriteResponse(c, errors.WithCode(code.ErrDatabase, err.Error()), nil)

		return
	}

	// only update expires and description
	// 更新secret的到期时间和描述信息
	secret.Expires = r.Expires
	secret.Description = r.Description
	secret.Extend = r.Extend

	// 校验secret数据
	if errs := secret.Validate(); len(errs) != 0 {
		core.WriteResponse(c, errors.WithCode(code.ErrValidation, errs.ToAggregate().Error()), nil)

		return
	}

	// 更新secret信息
	if err := s.srv.Secrets().Update(c, secret, metav1.UpdateOptions{}); err != nil {
		core.WriteResponse(c, err, nil)

		return
	}

	core.WriteResponse(c, nil, secret)
}
