/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package log

import (
	"fmt"
	"strings"

	"github.com/marmotedu/component-base/pkg/json"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	flagLevel             = "log.level"
	flagDisableCaller     = "log.disable-caller"
	flagDisableStacktrace = "log.disable-stacktrace"
	flagFormat            = "log.format"
	flagEnableColor       = "log.enable-color"
	flagOutputPaths       = "log.output-paths"
	flagErrorOutputPaths  = "log.error-output-paths"
	flagDevelopment       = "log.development"
	flagName              = "log.name"
	// 日志输出格式
	consoleFormat = "console"
	jsonFormat    = "json"
)

// Options contains configuration items related to log.
// Options包含日志相关的配置
type Options struct {
	OutputPaths       []string `json:"output-paths"       mapstructure:"output-paths"`       // 日志输出路径
	ErrorOutputPaths  []string `json:"error-output-paths" mapstructure:"error-output-paths"` // 错误输出路径
	Level             string   `json:"level"              mapstructure:"level"`              // 启用的日志等级
	Format            string   `json:"format"             mapstructure:"format"`             // 日志格式，只能是`console`或`json`
	DisableCaller     bool     `json:"disable-caller"     mapstructure:"disable-caller"`     // 是否要禁用caller
	DisableStacktrace bool     `json:"disable-stacktrace" mapstructure:"disable-stacktrace"` // 是否禁用栈追踪
	EnableColor       bool     `json:"enable-color"       mapstructure:"enable-color"`       // 是否启用颜色
	Development       bool     `json:"development"        mapstructure:"development"`        // 是否使用开发模式
	Name              string   `json:"name"               mapstructure:"name"`               // 日志名称
}

// NewOptions creates an Options object with default parameters.
// NewOptions 创建一个带有默认参数的Options对象
func NewOptions() *Options {
	return &Options{
		Level:             zapcore.InfoLevel.String(), // 默认日志level是info
		DisableCaller:     false,                      // 不禁用Caller
		DisableStacktrace: false,                      // 不禁用栈追踪
		Format:            consoleFormat,              // 以文本格式输出到控制台
		EnableColor:       false,                      // 不启用颜色
		Development:       false,                      // 不使用开发模式
		OutputPaths:       []string{"stdout"},         // 默认输出路径为stdout
		ErrorOutputPaths:  []string{"stderr"},         // 默认错误输出为stderr
	}
}

// Validate validate the options fields.
// Validate 验证选项中的字段
func (o *Options) Validate() []error {
	var errs []error
	// 验证选项中的Level字段是否合法，通过zapcore.Level的UnmarshalText方法把Options中的Level反解析出来，如果内容合法则可以解析成功。
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		errs = append(errs, err)
	}
	// 验证选项中的Format字段，只能是
	format := strings.ToLower(o.Format)
	if format != consoleFormat && format != jsonFormat {
		errs = append(errs, fmt.Errorf("not a valid log format: %q", o.Format))
	}

	return errs
}

// AddFlags adds flags for log to the specified FlagSet object.
// AddFlags 增加日志的flags到指定的FlagSet对象中
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Level, flagLevel, o.Level, "Minimum log output `LEVEL`.")
	fs.BoolVar(&o.DisableCaller, flagDisableCaller, o.DisableCaller, "Disable output of caller information in the log.")
	fs.BoolVar(&o.DisableStacktrace, flagDisableStacktrace,
		o.DisableStacktrace, "Disable the log to record a stack trace for all messages at or above panic level.")
	fs.StringVar(&o.Format, flagFormat, o.Format, "Log output `FORMAT`, support plain or json format.")
	fs.BoolVar(&o.EnableColor, flagEnableColor, o.EnableColor, "Enable output ansi colors in plain format logs.")
	fs.StringSliceVar(&o.OutputPaths, flagOutputPaths, o.OutputPaths, "Output paths of log.")
	fs.StringSliceVar(&o.ErrorOutputPaths, flagErrorOutputPaths, o.ErrorOutputPaths, "Error output paths of log.")
	fs.BoolVar(
		&o.Development,
		flagDevelopment,
		o.Development,
		"Development puts the logger in development mode, which changes "+
			"the behavior of DPanicLevel and takes stacktraces more liberally.",
	)
	fs.StringVar(&o.Name, flagName, o.Name, "The name of the logger.")
}

// String 把Options对象序列化成字符串
func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}

// Build constructs a global zap logger from the Config and Options.
// Build 根据配置和选项构建一个全局的zap looger
func (o *Options) Build() error {
	// 验证options的Level字段，如果不合法设置成Info级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(o.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	// 设置LevelEncoder，默认把日志级别输出为大写，如果Format为`console`且启用了颜色则输出为带有颜色的大写
	encodeLevel := zapcore.CapitalLevelEncoder
	if o.Format == consoleFormat && o.EnableColor {
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}
	// 创建Logger的配置
	zc := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapLevel), // 设置动态Level
		Development:       o.Development,                  // 设置运行模式
		DisableCaller:     o.DisableCaller,                // 是否禁用Caller
		DisableStacktrace: o.DisableStacktrace,            // 是否禁用栈追踪
		Sampling: &zap.SamplingConfig{ // 设置日志的采样策略
			Initial:    100, // 初始采集100条日志
			Thereafter: 100, // 之后每次采集100条日志
		},
		Encoding: o.Format, // 日志的编码格式，只能是`json`或`console`
		EncoderConfig: zapcore.EncoderConfig{ // 日志输出时的编码配置
			MessageKey:     "message",                   // 日志信息的key
			LevelKey:       "level",                     // 日志等级信息的key
			TimeKey:        "timestamp",                 // 日志中时间戳的key
			NameKey:        "logger",                    // logger名称的key
			CallerKey:      "caller",                    // 调用者的key
			StacktraceKey:  "stacktrace",                // 栈追踪的key
			LineEnding:     zapcore.DefaultLineEnding,   // 设置日志的换行符
			EncodeLevel:    encodeLevel,                 // 设置LevelEncoder用于输出日志等级时的编码方式
			EncodeTime:     timeEncoder,                 // 设置输出时间戳时的编码方式，自定义时间格式
			EncodeDuration: milliSecondsDurationEncoder, // 设置输出持续时间的编码方式
			EncodeCaller:   zapcore.ShortCallerEncoder,  // 设置输出调用者信息格式的编码方式
			EncodeName:     zapcore.FullNameEncoder,     // 设置logger名称的编码方式
		},
		OutputPaths:      o.OutputPaths,      // 日志输出路径
		ErrorOutputPaths: o.ErrorOutputPaths, // 错误日志输出路径
	}
	logger, err := zc.Build(zap.AddStacktrace(zapcore.PanicLevel)) // 创建Logger时设置只有发生panic级别的错误时才进行栈追踪
	if err != nil {
		return err
	}
	zap.RedirectStdLog(logger.Named(o.Name)) // 把标准库log的输出日志重定向到带有名称的子logger中
	zap.ReplaceGlobals(logger)               // 把logger设置为全局logger

	return nil
}
