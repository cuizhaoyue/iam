// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package secret

import (
	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/core"
	metav1 "github.com/marmotedu/component-base/pkg/meta/v1"

	"github.com/marmotedu/iam/internal/pkg/middleware"
	"github.com/marmotedu/iam/pkg/log"
)

// Delete delete a secret by the secret identifier.
func (s *SecretController) Delete(c *gin.Context) {
	log.L(c).Info("delete secret function called.")
	opts := metav1.DeleteOptions{Unscoped: true} // 设置永久删除
	// 从path参数中获取secret名称，通过用户名和secret名称删除数据库中的secret数据
	if err := s.srv.Secrets().Delete(c, c.GetString(middleware.UsernameKey), c.Param("name"), opts); err != nil {
		core.WriteResponse(c, err, nil)

		return
	}

	core.WriteResponse(c, nil, nil)
}
