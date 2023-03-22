// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package options contains flags and options for initializing an apiserver
package options

import (
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/marmotedu/component-base/pkg/json"

	genericoptions "github.com/marmotedu/iam/internal/pkg/options"
	"github.com/marmotedu/iam/internal/pump/analytics"
	"github.com/marmotedu/iam/pkg/log"
)

// PumpConfig defines options for pump back-end.
// 定义了pump的后端配置，Meta为自定义配置，其它为通用配置。
// 通用配置可以配置共享，减少开发的维护的工作量，自定义配置可以适配不同的pump的差异化配置
type PumpConfig struct {
	Type                  string                     `json:"type"                    mapstructure:"type"` // pump 类型
	Filters               analytics.AnalyticsFilters `json:"filters"                 mapstructure:"filters"`
	Timeout               int                        `json:"timeout"                 mapstructure:"timeout"`
	OmitDetailedRecording bool                       `json:"omit-detailed-recording" mapstructure:"omit-detailed-recording"`
	Meta                  map[string]interface{}     `json:"meta"                    mapstructure:"meta"`
}

// Options runs a pumpserver. 运行pump服务的配置
type Options struct {
	PurgeDelay            int                          `json:"purge-delay"             mapstructure:"purge-delay"`             // 审计日志清理时间间隔，默认10s
	Pumps                 map[string]PumpConfig        `json:"pumps"                   mapstructure:"pumps"`                   // pump 配置
	HealthCheckPath       string                       `json:"health-check-path"       mapstructure:"health-check-path"`       // 健康检查路由，默认为 /healthz
	HealthCheckAddress    string                       `json:"health-check-address"    mapstructure:"health-check-address"`    // 健康检查绑定端口，默认为 0.0.0.0:7070
	OmitDetailedRecording bool                         `json:"omit-detailed-recording" mapstructure:"omit-detailed-recording"` // 设置为 true 会记录详细的授权审计日志，默认为 false
	RedisOptions          *genericoptions.RedisOptions `json:"redis"                   mapstructure:"redis"`                   // Redis 配置
	Log                   *log.Options                 `json:"log"                     mapstructure:"log"`                     // 日志配置
}

// NewOptions creates a new Options object with default parameters.
func NewOptions() *Options {
	s := Options{
		PurgeDelay: 10,
		Pumps: map[string]PumpConfig{
			"csv": {
				Type: "csv",
				Meta: map[string]interface{}{
					"csv_dir": "./analytics-data",
				},
			},
		},
		HealthCheckPath:    "healthz",
		HealthCheckAddress: "0.0.0.0:7070",
		RedisOptions:       genericoptions.NewRedisOptions(),
		Log:                log.NewOptions(),
	}

	return &s
}

// Flags returns flags for a specific APIServer by section name.
func (o *Options) Flags() (fss cliflag.NamedFlagSets) {
	o.RedisOptions.AddFlags(fss.FlagSet("redis"))
	o.Log.AddFlags(fss.FlagSet("logs"))

	// Note: the weird ""+ in below lines seems to be the only way to get gofmt to
	// arrange these text blocks sensibly. Grrr.
	fs := fss.FlagSet("misc")
	fs.IntVar(&o.PurgeDelay, "purge-delay", o.PurgeDelay, ""+
		"This setting the purge delay (in seconds) when purge the data from Redis to MongoDB or other data stores.")
	fs.StringVar(&o.HealthCheckPath, "health-check-path", o.HealthCheckPath, ""+
		"Specifies liveness health check request path.")
	fs.StringVar(&o.HealthCheckAddress, "health-check-address", o.HealthCheckAddress, ""+
		"Specifies liveness health check bind address.")
	fs.BoolVar(&o.OmitDetailedRecording, "omit-detailed-recording", o.OmitDetailedRecording, ""+
		"Setting this to true will avoid writing policy fields for each authorization request in pumps.")

	return fss
}

func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}
