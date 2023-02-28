// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package app

import (
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
)

// CliOptions abstracts configuration options for reading parameters from the
// command line.
// CliOptions抽取了从命令行读取的配置选项。
type CliOptions interface {
	// AddFlags adds flags to the specified FlagSet object.
	// AddFlags(fs *pflag.FlagSet)
	Flags() (fss cliflag.NamedFlagSets)
	Validate() []error
}

// ConfigurableOptions abstracts configuration options for reading parameters
// from a configuration file.
// ConfigurableOptions 抽取了从配置文件读取到的参数的配置选项。
type ConfigurableOptions interface {
	// ApplyFlags parsing parameters from the command line or configuration file
	// to the options instance.
	ApplyFlags() []error
}

// CompleteableOptions abstracts options which can be completed.
// CompleteableOptions 抽象可以被补全的选项。
type CompleteableOptions interface {
	// Complete 通过配置补全，可以确保一些重要的配置项具有默认值，
	// 当这些配置项没有被配置时，程序也仍然能够正常启动。
	// 一个大型项目，有很多配置项，我们不可能对每一个配置项都进行配置。
	// 所以，给重要配置项设置默认值，就显得很重要了。
	Complete() error
}

// PrintableOptions abstracts options which can be printed.
// PrintableOptions抽象可以被打印的选项。
type PrintableOptions interface {
	String() string
}
