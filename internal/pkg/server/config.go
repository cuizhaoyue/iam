// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package server

import (
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/util/homedir"
	"github.com/spf13/viper"

	"github.com/marmotedu/iam/pkg/log"
)

/*
服务创建基本流程：

	config -> completedConfig 初始化操作 -> genericapiserver -> Run -> Close
*/
const (
	// RecommendedHomeDir defines the default directory used to place all iam service configurations.
	// 定义默认存放服务配置文件的目录
	RecommendedHomeDir = ".iam"

	// RecommendedEnvPrefix defines the ENV prefix used by all iam service.
	// 定义iam服务使用的环境变量的前缀
	RecommendedEnvPrefix = "IAM"
)

// Config is a structure used to configure a GenericAPIServer.
// Its members are sorted roughly in order of importance for composers.
// Config是HTTP服务器的配置
type Config struct {
	SecureServing   *SecureServingInfo
	InsecureServing *InsecureServingInfo
	Jwt             *JwtInfo
	Mode            string   // 服务运行模式, debug或release
	Middlewares     []string // 要加载的中间件
	Healthz         bool     // 启动健康检查
	EnableProfiling bool     // 是否启动性能分析
	EnableMetrics   bool     // 是否公开metric
}

// CertKey contains configuration items related to certificate.
// CertKey包含证书相关的配置项。
type CertKey struct {
	// CertFile is a file containing a PEM-encoded certificate, and possibly the complete certificate chain
	CertFile string
	// KeyFile is a file containing a PEM-encoded private key for the certificate specified by CertFile
	KeyFile string
}

// SecureServingInfo holds configuration of the TLS server.
// 保存TLS服务的配置
type SecureServingInfo struct {
	BindAddress string
	BindPort    int
	CertKey     CertKey // 证书信息
}

// Address join host IP address and host port number into an address string, like: 0.0.0.0:8443.
// Address连接主机ip和端口
func (s *SecureServingInfo) Address() string {
	return net.JoinHostPort(s.BindAddress, strconv.Itoa(s.BindPort))
}

// InsecureServingInfo holds configuration of the insecure http server.
// http服务的配置
type InsecureServingInfo struct {
	Address string
}

// JwtInfo defines jwt fields used to create jwt authentication middleware.
// 定义了jwt字段用来创建jwt认证中间件
type JwtInfo struct {
	// defaults to "iam jwt"
	Realm string // 服务器返回的realm，一般是域名
	// defaults to empty
	Key string
	// defaults to one hour
	Timeout time.Duration // 超时时间，默认1h
	// defaults to zero
	MaxRefresh time.Duration // 刷新时间，默认0不刷新
}

// NewConfig returns a Config struct with the default values.
// 创建一个带有默认值的配置对象
func NewConfig() *Config {
	return &Config{
		Healthz:         true,
		Mode:            gin.ReleaseMode,
		Middlewares:     []string{},
		EnableProfiling: true,
		EnableMetrics:   true,
		Jwt: &JwtInfo{
			Realm:      "iam jwt",
			Timeout:    1 * time.Hour,
			MaxRefresh: 1 * time.Hour,
		},
	}
}

// CompletedConfig is the completed configuration for GenericAPIServer.
// CompleteConfig 是 GenericAPIServer 的完整配置
type CompletedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields. If you're going to `ApplyOptions`, do that first. It's mutating the receiver.
// 对通用服务配置进行补全操作
func (c *Config) Complete() CompletedConfig {
	return CompletedConfig{c}
}

// New returns a new instance of GenericAPIServer from the given config.
// 根据补全后的配置构建通用apiserver服务
func (c CompletedConfig) New() (*GenericAPIServer, error) {
	// setMode before gin.New()
	gin.SetMode(c.Mode) // 设置服务启动模式

	s := &GenericAPIServer{ // 根据补全的服务配置创建REST API SERVER实例
		SecureServingInfo:   c.SecureServing,
		InsecureServingInfo: c.InsecureServing,
		healthz:             c.Healthz,
		enableMetrics:       c.EnableMetrics,
		enableProfiling:     c.EnableProfiling,
		middlewares:         c.Middlewares,
		Engine:              gin.New(),
	}

	initGenericAPIServer(s) // 初始化API SERVER实例

	// 返回初始化后的通用api server实例
	return s, nil
}

// LoadConfig reads in config file and ENV variables if set.
// 读取配置文件和加载环境变量
func LoadConfig(cfg string, defaultName string) {
	if cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(homedir.HomeDir(), RecommendedHomeDir))
		viper.AddConfigPath("/etc/iam")
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
