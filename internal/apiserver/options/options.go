// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package options contains flags and options for initializing an apiserver
package options

import (
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/marmotedu/component-base/pkg/json"
	"github.com/marmotedu/component-base/pkg/util/idutil"
	genericoptions "github.com/marmotedu/iam/internal/pkg/options"
	"github.com/marmotedu/iam/internal/pkg/server"
	"github.com/marmotedu/iam/pkg/log"
)

// Options runs an iam api server.
// Options配置：用来构建命令行参数，它的值来自于命令行选项或者配置文件（也可能是二者 Merge 后的配置）。Options 可以用来构建应用框架，Options 配置也是应用配置的输入。
type Options struct {
	GenericServerRunOptions *genericoptions.ServerRunOptions       `json:"server"   mapstructure:"server"`
	GRPCOptions             *genericoptions.GRPCOptions            `json:"grpc"     mapstructure:"grpc"`
	InsecureServing         *genericoptions.InsecureServingOptions `json:"insecure" mapstructure:"insecure"`
	SecureServing           *genericoptions.SecureServingOptions   `json:"secure"   mapstructure:"secure"`
	MySQLOptions            *genericoptions.MySQLOptions           `json:"mysql"    mapstructure:"mysql"`
	RedisOptions            *genericoptions.RedisOptions           `json:"redis"    mapstructure:"redis"`
	JwtOptions              *genericoptions.JwtOptions             `json:"jwt"      mapstructure:"jwt"`
	Log                     *log.Options                           `json:"log"      mapstructure:"log"`
	FeatureOptions          *genericoptions.FeatureOptions         `json:"feature"  mapstructure:"feature"`
}

// NewOptions creates a new Options object with default parameters.
// NewOptions 创建一个新的带有默认选项的Options对象
func NewOptions() *Options { // Options用来构建命令行参数
	o := Options{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),       // 通用服务运行的配置选项
		GRPCOptions:             genericoptions.NewGRPCOptions(),            // grpc服务的配置选项选项
		InsecureServing:         genericoptions.NewInsecureServingOptions(), // http服务的配置选项
		SecureServing:           genericoptions.NewSecureServingOptions(),   // HTTPS服务的配置选项
		MySQLOptions:            genericoptions.NewMySQLOptions(),           // 连接mysql实例的配置选项
		RedisOptions:            genericoptions.NewRedisOptions(),           // 连接redis实例的配置选项
		JwtOptions:              genericoptions.NewJwtOptions(),             // jwt相关的选项
		Log:                     log.NewOptions(),                           // 创建Logger的配置项
		FeatureOptions:          genericoptions.NewFeatureOptions(),         // server功能的配置
	}

	return &o
}

// ApplyTo applies the run options to the method receiver and returns self.
func (o *Options) ApplyTo(c *server.Config) error {
	return nil
}

// Flags returns flags for a specific APIServer by section name.
// Flags 对Flag进行分组。把Options选项的配置加入到各个命名的FlagSet中。
func (o *Options) Flags() (fss cliflag.NamedFlagSets) {
	o.GenericServerRunOptions.AddFlags(fss.FlagSet("generic"))
	o.JwtOptions.AddFlags(fss.FlagSet("jwt"))
	o.GRPCOptions.AddFlags(fss.FlagSet("grpc"))
	o.MySQLOptions.AddFlags(fss.FlagSet("mysql"))
	o.RedisOptions.AddFlags(fss.FlagSet("redis"))
	o.FeatureOptions.AddFlags(fss.FlagSet("features"))
	o.InsecureServing.AddFlags(fss.FlagSet("insecure serving"))
	o.SecureServing.AddFlags(fss.FlagSet("secure serving"))
	o.Log.AddFlags(fss.FlagSet("logs"))

	return fss
}

// 配置序列化
func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

// Complete set default Options.
// 通过配置补全，可以确保一些重要的配置项具有默认值，
// 当这些配置项没有被配置时，程序也仍然能够正常启动。
// 一个大型项目，有很多配置项，我们不可能对每一个配置项都进行配置。
// 所以，给重要配置项设置默认值，就显得很重要了。
func (o *Options) Complete() error {
	if o.JwtOptions.Key == "" {
		o.JwtOptions.Key = idutil.NewSecretKey()
	}

	return o.SecureServing.Complete()
}
