// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package authzserver

import (
	"context"

	"github.com/marmotedu/errors"

	"github.com/marmotedu/iam/internal/authzserver/analytics"
	"github.com/marmotedu/iam/internal/authzserver/config"
	"github.com/marmotedu/iam/internal/authzserver/load"
	"github.com/marmotedu/iam/internal/authzserver/load/cache"
	"github.com/marmotedu/iam/internal/authzserver/store/apiserver"
	genericoptions "github.com/marmotedu/iam/internal/pkg/options"
	genericapiserver "github.com/marmotedu/iam/internal/pkg/server"
	"github.com/marmotedu/iam/pkg/log"
	"github.com/marmotedu/iam/pkg/shutdown"
	"github.com/marmotedu/iam/pkg/shutdown/shutdownmanagers/posixsignal"
	"github.com/marmotedu/iam/pkg/storage"
)

// RedisKeyPrefix defines the prefix key in redis for analytics data.
const RedisKeyPrefix = "analytics-"

type authzServer struct {
	gs               *shutdown.GracefulShutdown
	rpcServer        string
	clientCA         string
	redisOptions     *genericoptions.RedisOptions
	genericAPIServer *genericapiserver.GenericAPIServer
	analyticsOptions *analytics.AnalyticsOptions
	redisCancelFunc  context.CancelFunc
}

type preparedAuthzServer struct {
	*authzServer
}

// func createAuthzServer(cfg *config.Config) (*authzServer, error) {.
func createAuthzServer(cfg *config.Config) (*authzServer, error) {
	gs := shutdown.New()
	gs.AddShutdownManager(posixsignal.NewPosixSignalManager())

	genericConfig, err := buildGenericConfig(cfg)
	if err != nil {
		return nil, err
	}

	genericServer, err := genericConfig.Complete().New()
	if err != nil {
		return nil, err
	}

	server := &authzServer{
		gs:               gs,
		redisOptions:     cfg.RedisOptions,
		analyticsOptions: cfg.AnalyticsOptions,
		rpcServer:        cfg.RPCServer,
		clientCA:         cfg.ClientCA,
		genericAPIServer: genericServer,
	}

	return server, nil
}

// PrepareRun 应用初始化
func (s *authzServer) PrepareRun() preparedAuthzServer {
	// 初始化，包括保持redis连接、创建缓存实例、启动密钥和策略的同步工作、开启analytics服务
	_ = s.initialize()

	initRouter(s.genericAPIServer.Engine)

	return preparedAuthzServer{s}
}

// Run start to run AuthzServer. 运行服务
func (s preparedAuthzServer) Run() error {
	stopCh := make(chan struct{})

	// start shutdown managers
	if err := s.gs.Start(); err != nil {
		log.Fatalf("start shutdown manager failed: %s", err.Error())
	}

	//nolint: errcheck
	go s.genericAPIServer.Run() // 启动http服务

	// in order to ensure that the reported data is not lost,
	// please ensure the following graceful shutdown sequence
	// 为了保证数据不丢失，要保证下面的优雅关闭服务的顺序.
	s.gs.AddShutdownCallback(shutdown.ShutdownFunc(func(string) error {
		s.genericAPIServer.Close()
		if s.analyticsOptions.Enable {
			analytics.GetAnalytics().Stop()
		}
		s.redisCancelFunc()

		return nil
	}))

	// blocking here via channel to prevents the process exit.
	<-stopCh

	return nil
}

func buildGenericConfig(cfg *config.Config) (genericConfig *genericapiserver.Config, lastErr error) {
	genericConfig = genericapiserver.NewConfig()
	if lastErr = cfg.GenericServerRunOptions.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	if lastErr = cfg.FeatureOptions.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	if lastErr = cfg.SecureServing.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	if lastErr = cfg.InsecureServing.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	return
}

// 构建redis的配置
func (s *authzServer) buildStorageConfig() *storage.Config {
	return &storage.Config{
		Host:                  s.redisOptions.Host,
		Port:                  s.redisOptions.Port,
		Addrs:                 s.redisOptions.Addrs,
		MasterName:            s.redisOptions.MasterName,
		Username:              s.redisOptions.Username,
		Password:              s.redisOptions.Password,
		Database:              s.redisOptions.Database,
		MaxIdle:               s.redisOptions.MaxIdle,
		MaxActive:             s.redisOptions.MaxActive,
		Timeout:               s.redisOptions.Timeout,
		EnableCluster:         s.redisOptions.EnableCluster,
		UseSSL:                s.redisOptions.UseSSL,
		SSLInsecureSkipVerify: s.redisOptions.SSLInsecureSkipVerify,
	}
}

func (s *authzServer) initialize() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.redisCancelFunc = cancel

	// keep redis connected 保持和redis的连接状，断开会重新连接
	go storage.ConnectToRedis(ctx, s.buildStorageConfig())

	// cron to reload all secrets and policies from iam-apiserver
	// 创建缓存实例，定时从iam-apiserver中同步secret和policy过来
	cacheIns, err := cache.GetCacheInsOr(apiserver.GetAPIServerFactoryOrDie(s.rpcServer, s.clientCA))
	if err != nil {
		return errors.Wrap(err, "get cache instance failed")
	}

	// load包完成密钥和策略的缓存
	load.NewLoader(ctx, cacheIns).Start()

	// start analytics service 开启analytics服务
	if s.analyticsOptions.Enable {
		analyticsStore := storage.RedisCluster{KeyPrefix: RedisKeyPrefix}           // analytics服务使用的redis存储实例
		analyticsIns := analytics.NewAnalytics(s.analyticsOptions, &analyticsStore) // 创建analytics实例
		analyticsIns.Start()                                                        // 启动analytics服务
	}

	return nil
}
