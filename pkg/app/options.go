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
	Complete() error
}

// PrintableOptions abstracts options which can be printed.
// PrintableOptions抽象可以被打印的选项。
type PrintableOptions interface {
	String() string
}
