// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/core"
	"github.com/marmotedu/errors"

	"github.com/marmotedu/iam/internal/apiserver/controller/v1/policy"
	"github.com/marmotedu/iam/internal/apiserver/controller/v1/secret"
	"github.com/marmotedu/iam/internal/apiserver/controller/v1/user"
	"github.com/marmotedu/iam/internal/apiserver/store/mysql"
	"github.com/marmotedu/iam/internal/pkg/code"
	"github.com/marmotedu/iam/internal/pkg/middleware"
	"github.com/marmotedu/iam/internal/pkg/middleware/auth"

	// custom gin validators.
	_ "github.com/marmotedu/iam/pkg/validator"
)

func initRouter(g *gin.Engine) {
	installMiddleware(g) // 安装中间件
	installController(g) // 安装控制器
}

func installMiddleware(g *gin.Engine) {
}

func installController(g *gin.Engine) *gin.Engine {
	// Middlewares.
	jwtStrategy, _ := newJWTAuth().(auth.JWTStrategy) // 创建jwt认证策略
	g.POST("/login", jwtStrategy.LoginHandler)        // 登录路由
	g.POST("/logout", jwtStrategy.LogoutHandler)      // 登出路由
	// Refresh time can be longer than token timeout
	g.POST("/refresh", jwtStrategy.RefreshHandler) // 刷新路由

	// auto 策略: 该策略会根据 HTTP 头Authorization: Basic XX.YY.ZZ和Authorization: Bearer XX.YY.ZZ自动选择使用 Basic 认证还是 Bearer 认证。
	auto := newAutoAuth()
	g.NoRoute(auto.AuthFunc(), func(c *gin.Context) { // 路由不存在时的处理函数
		core.WriteResponse(c, errors.WithCode(code.ErrPageNotFound, "Page not found."), nil)
	})

	// v1 handlers, requiring authentication
	storeIns, _ := mysql.GetMySQLFactoryOr(nil) // 获取存储实例
	v1 := g.Group("/v1")
	{
		// user RESTful resource
		userv1 := v1.Group("/users")
		{
			userController := user.NewUserController(storeIns) // 控制层，用户处理器

			userv1.POST("", userController.Create) // 创建用户
			userv1.Use(auto.AuthFunc(), middleware.Validation())
			// v1.PUT("/find_password", userController.FindPassword)
			userv1.DELETE("", userController.DeleteCollection)                 // admin api，删除用户集合
			userv1.DELETE(":name", userController.Delete)                      // admin api，删除单个用户
			userv1.PUT(":name/change-password", userController.ChangePassword) // 修改用户
			userv1.PUT(":name", userController.Update)                         // 更新用户信息
			userv1.GET("", userController.List)                                // 列出用户信息
			userv1.GET(":name", userController.Get)                            // admin api，获取用户信息
		}

		v1.Use(auto.AuthFunc()) // 添加认证中间件

		// policy RESTful resource
		policyv1 := v1.Group("/policies", middleware.Publish())
		{
			policyController := policy.NewPolicyController(storeIns)

			policyv1.POST("", policyController.Create)
			policyv1.DELETE("", policyController.DeleteCollection)
			policyv1.DELETE(":name", policyController.Delete)
			policyv1.PUT(":name", policyController.Update)
			policyv1.GET("", policyController.List)
			policyv1.GET(":name", policyController.Get)
		}

		// secret RESTful resource
		secretv1 := v1.Group("/secrets", middleware.Publish())
		{
			secretController := secret.NewSecretController(storeIns)

			secretv1.POST("", secretController.Create)
			secretv1.DELETE(":name", secretController.Delete)
			secretv1.PUT(":name", secretController.Update)
			secretv1.GET("", secretController.List)
			secretv1.GET(":name", secretController.Get)
		}
	}

	return g
}
