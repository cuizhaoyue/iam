// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	v1 "github.com/marmotedu/api/apiserver/v1"
	metav1 "github.com/marmotedu/component-base/pkg/meta/v1"
	"github.com/spf13/viper"

	"github.com/marmotedu/iam/internal/apiserver/store"
	"github.com/marmotedu/iam/internal/pkg/middleware"
	"github.com/marmotedu/iam/internal/pkg/middleware/auth"
	"github.com/marmotedu/iam/pkg/log"
)

const (
	// APIServerAudience defines the value of jwt audience field. // jwt观众字段
	APIServerAudience = "iam.api.marmotedu.com"

	// APIServerIssuer defines the value of jwt issuer field. jwt发行者字段
	APIServerIssuer = "iam-apiserver"
)

// 登录信息
type loginInfo struct {
	Username string `form:"username" json:"username" binding:"required,username"`
	Password string `form:"password" json:"password" binding:"required,password"`
}

// 创建basic认证策略
func newBasicAuth() middleware.AuthStrategy {
	return auth.NewBasicStrategy(func(username string, password string) bool {
		// fetch user from database 从数据库中获取用户信息
		user, err := store.Client().Users().Get(context.TODO(), username, metav1.GetOptions{})
		if err != nil {
			return false
		}

		// Compare the login password with the user password. 比较登录密码和数据库中存储的用户密码
		if err := user.Compare(password); err != nil {
			return false
		}

		user.LoginedAt = time.Now()
		_ = store.Client().Users().Update(context.TODO(), user, metav1.UpdateOptions{})

		return true
	})
}

// 创建jwt认证策略
func newJWTAuth() middleware.AuthStrategy {
	ginjwt, _ := jwt.New(&jwt.GinJWTMiddleware{ // jwt中间件
		Realm:            viper.GetString("jwt.Realm"),         // 展示给用户的realm名称
		SigningAlgorithm: "HS256",                              // 签名算法
		Key:              []byte(viper.GetString("jwt.key")),   // 用于签名的secretKey
		Timeout:          viper.GetDuration("jwt.timeout"),     // jwt token的有效持续时间，默认1小时
		MaxRefresh:       viper.GetDuration("jwt.max-refresh"), // jwt token的更新时间，默认为0不允许更新
		Authenticator:    authenticator(),                      // 执行认证操作的回调函数
		LoginResponse:    loginResponse(),                      // 用户登录后的回调函数
		LogoutResponse: func(c *gin.Context, code int) { // 用户退出后的回调函数
			c.JSON(http.StatusOK, nil)
		},
		RefreshResponse: refreshResponse(), // 更新token后的返回函数
		PayloadFunc:     payloadFunc(),     // 登录的时候会调用，用于添加额外的数据，发起请求时通过c.Get(JWT_PAYLOAD)可以使用
		IdentityHandler: func(c *gin.Context) interface{} { // 设置识别函数，用于识别是否已经通过认证，它的返回值是Authorizator函数中的传入data
			claims := jwt.ExtractClaims(c) // 提取MapClaims

			return claims[jwt.IdentityKey]
		},
		IdentityKey:  middleware.UsernameKey, // 设置identity key
		Authorizator: authorizator(),         // 对已认证用户执行授权操作的回调函数。在认证通过后调用
		Unauthorized: func(c *gin.Context, code int, message string) { // 定义未授权时的操作函数
			c.JSON(code, gin.H{
				"message": message,
			})
		},
		TokenLookup:   "header: Authorization, query: token, cookie: jwt", // 一种指定格式的字符串，用于从request中抽取token
		TokenHeadName: "Bearer",                                           // header中的字符串，默认为Bearer
		SendCookie:    true,                                               // 作为cookie返回token
		TimeFunc:      time.Now,                                           // 提供当前时间的函数
		// TODO: HTTPStatusMessageFunc:
	})

	return auth.NewJWTStrategy(*ginjwt)
}

func newAutoAuth() middleware.AuthStrategy {
	return auth.NewAutoStrategy(newBasicAuth().(auth.BasicStrategy), newJWTAuth().(auth.JWTStrategy))
}

// 返回用于执行认证的回调函数
func authenticator() func(c *gin.Context) (interface{}, error) {
	return func(c *gin.Context) (interface{}, error) {
		var login loginInfo
		var err error

		// support header and body both 同时支持header和body
		if c.Request.Header.Get("Authorization") != "" { // 从header中获取认证信息
			login, err = parseWithHeader(c) // 解析出username和password
		} else { // 从body中获取认证信息
			login, err = parseWithBody(c) // 从body中解析username和password
		}
		if err != nil {
			return "", jwt.ErrFailedAuthentication
		}

		// Get the user information by the login username. 通过用户名从数据库中获取user对象
		user, err := store.Client().Users().Get(c, login.Username, metav1.GetOptions{})
		if err != nil {
			log.Errorf("get user information failed: %s", err.Error())

			return "", jwt.ErrFailedAuthentication
		}

		// Compare the login password with the user password. 比较登录密码和用户密码是否一致
		if err := user.Compare(login.Password); err != nil {
			return "", jwt.ErrFailedAuthentication
		}

		user.LoginedAt = time.Now()                                        // 更新登录时间
		_ = store.Client().Users().Update(c, user, metav1.UpdateOptions{}) // 更新user数据

		return user, nil
	}
}

// 解析从头部中获取的认证信息
func parseWithHeader(c *gin.Context) (loginInfo, error) {
	auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2) // 把认证信息"Basic xxxxxxxxx"分割成两段
	if len(auth) != 2 || auth[0] != "Basic" {                             // 确保是Basic模式认证
		log.Errorf("get basic string from Authorization header failed")

		return loginInfo{}, jwt.ErrFailedAuthentication
	}

	payload, err := base64.StdEncoding.DecodeString(auth[1]) // base64反解码得到username:password
	if err != nil {
		log.Errorf("decode basic string: %s", err.Error())

		return loginInfo{}, jwt.ErrFailedAuthentication
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		log.Errorf("parse payload failed")

		return loginInfo{}, jwt.ErrFailedAuthentication
	}

	return loginInfo{
		Username: pair[0],
		Password: pair[1],
	}, nil
}

// 从body中解析username和password
func parseWithBody(c *gin.Context) (loginInfo, error) {
	var login loginInfo
	if err := c.ShouldBindJSON(&login); err != nil {
		log.Errorf("parse login parameters: %s", err.Error())

		return loginInfo{}, jwt.ErrFailedAuthentication
	}

	return login, nil
}

func refreshResponse() func(c *gin.Context, code int, token string, expire time.Time) {
	return func(c *gin.Context, code int, token string, expire time.Time) {
		c.JSON(http.StatusOK, gin.H{
			"token":  token,
			"expire": expire.Format(time.RFC3339),
		})
	}
}

func loginResponse() func(c *gin.Context, code int, token string, expire time.Time) {
	return func(c *gin.Context, code int, token string, expire time.Time) {
		c.JSON(http.StatusOK, gin.H{ // 返回token和token的到期时间
			"token":  token,
			"expire": expire.Format(time.RFC3339),
		})
	}
}

func payloadFunc() func(data interface{}) jwt.MapClaims {
	return func(data interface{}) jwt.MapClaims {
		claims := jwt.MapClaims{
			"iss": APIServerIssuer,   // jwt token签发人
			"aud": APIServerAudience, // 接收jwt token的一方
		}
		if u, ok := data.(*v1.User); ok {
			claims[jwt.IdentityKey] = u.Name
			claims["sub"] = u.Name // 主题，可以鉴别用户
		}

		return claims
	}
}

func authorizator() func(data interface{}, c *gin.Context) bool {
	return func(data interface{}, c *gin.Context) bool {
		if v, ok := data.(string); ok {
			log.L(c).Infof("user `%s` is authenticated.", v)

			return true
		}

		return false
	}
}
