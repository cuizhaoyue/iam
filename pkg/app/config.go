// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package app

import (
	"fmt"
	"github.com/gosuri/uitable"
	"github.com/marmotedu/component-base/pkg/util/homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const configFlagName = "config"

var cfgFile string

// nolint: gochecknoinits
func init() { // 初始化中设置配置文件路径
	// 通过 addConfigFlag 调用，添加了 -c, --config FILE 命令行参数，用来指定配置文件
	pflag.StringVarP(&cfgFile, "config", "c", cfgFile, "Read configuration from specified `FILE`, "+
		"support JSON, TOML, YAML, HCL, or Java properties formats.")
}

// addConfigFlag adds flags for a specific server to the specified FlagSet
// object.
// addConfigFlag 添加config的flag到指定的FlagSet中，读取环境变量和配置文件
func addConfigFlag(basename string, fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(configFlagName))
	// 支持环境变量，通过 viper.SetEnvPrefix 来设置环境变量前缀，
	// 避免跟系统中的环境变量重名。通过 viper.SetEnvKeyReplacer 重写了 Env 键
	viper.AutomaticEnv() // 读取环境变量
	viper.SetEnvPrefix(strings.Replace(strings.ToUpper(basename), "-", "_", -1))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// 指定 Cobra Command 在执行命令之前，需要做的初始化工作
	cobra.OnInitialize(func() { // 执行命令前读取配置文件
		// 如果命令行参数中没有指定配置文件的路径，则加载默认路径下的配置文件，
		// 通过 viper.AddConfigPath、viper.SetConfigName 来设置配置文件搜索路径和配置文件名。
		// 通过设置默认的配置文件，可以使我们不用携带任何命令行参数，即可运行程序。
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile) // 指定了配置文件路径时直接读取配置文件
		} else {
			viper.AddConfigPath(".") // 添加配置文件的搜索路径

			if names := strings.Split(basename, "-"); len(names) > 1 {
				viper.AddConfigPath(filepath.Join(homedir.HomeDir(), "."+names[0]))
				viper.AddConfigPath(filepath.Join("/etc", names[0]))
			}

			viper.SetConfigName(basename) // 设置配置文件的名称
		}

		if err := viper.ReadInConfig(); err != nil { // 载入配置文件
			_, _ = fmt.Fprintf(os.Stderr, "Error: failed to read configuration file(%s): %v\n", cfgFile, err)
			os.Exit(1)
		}
	})
}

func printConfig() {
	if keys := viper.AllKeys(); len(keys) > 0 {
		fmt.Printf("%v Configuration items:\n", progressMessage)
		table := uitable.New()
		table.Separator = " "
		table.MaxColWidth = 80
		table.RightAlign(0)
		for _, k := range keys {
			table.AddRow(fmt.Sprintf("%s:", k), viper.Get(k))
		}
		fmt.Printf("%v", table)
	}
}

/*
// loadConfig reads in config file and ENV variables if set.
func loadConfig(cfg string, defaultName string) {
	if cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(homedir.HomeDir(), RecommendedHomeDir))
		viper.SetConfigName(defaultName)
	}

	// Use config file from the flag.
	viper.SetConfigType("yaml")              // set the type of the configuration to yaml.
	viper.AutomaticEnv()                     // read in environment variables that match.
	viper.SetEnvPrefix(RecommendedEnvPrefix) // set ENVIRONMENT variables prefix to IAM.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Warnf("WARNING: viper failed to discover and load the configuration file: %s", err.Error())
	}
}
*/
