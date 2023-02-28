// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package apiserver does all the work necessary to create an iam APIServer.
package apiserver

import (
	"github.com/marmotedu/iam/internal/apiserver/config"
	"github.com/marmotedu/iam/internal/apiserver/options"
	"github.com/marmotedu/iam/pkg/app"
	"github.com/marmotedu/iam/pkg/log"
)

const commandDesc = `The IAM API server validates and configures data
for the api objects which include users, policies, secrets, and
others. The API Server services REST operations to do the api objects management.

Find more iam-apiserver information at:
    https://github.com/marmotedu/iam/blob/master/docs/guide/en-US/cmd/iam-apiserver.md`

// NewApp creates an App object with default parameters.
func NewApp(basename string) *app.App {
	opts := options.NewOptions()                // 创建一个带有默认参数的Options配置，Options 配置是应用配置的输入
	application := app.NewApp("IAM API Server", // 创建应用，传入各应用配置
		basename,                         // basename:`iam-apiserver`
		app.WithOptions(opts),            // 传入Options配置，用于构建应用配置
		app.WithDescription(commandDesc), // 传入应用描述
		app.WithDefaultValidArgs(),       // 传入参数验证选项
		app.WithRunFunc(run(opts)),       // 注册启动函数
	)

	return application
}

// 定义apiserver的启动函数
func run(opts *options.Options) app.RunFunc {
	return func(basename string) error {
		log.Init(opts.Log) // 初始化日志配置（没有这一步就会使用默认的日志配置）
		defer log.Flush()

		cfg, err := config.CreateConfigFromOptions(opts) // 输入options获取到应用配置
		if err != nil {
			return err
		}

		return Run(cfg) // 启动apiserver
	}
}
