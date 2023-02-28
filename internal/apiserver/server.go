// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

/*
internal目录下是私有应用的包，apiserver包主要是apiserver服务的构建和运行
*/
package apiserver

import (
	"context"
	"fmt"

	pb "github.com/marmotedu/api/proto/apiserver/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/marmotedu/iam/internal/apiserver/config"
	cachev1 "github.com/marmotedu/iam/internal/apiserver/controller/v1/cache"
	"github.com/marmotedu/iam/internal/apiserver/store"
	"github.com/marmotedu/iam/internal/apiserver/store/mysql"
	genericoptions "github.com/marmotedu/iam/internal/pkg/options"
	genericapiserver "github.com/marmotedu/iam/internal/pkg/server"
	"github.com/marmotedu/iam/pkg/log"
	"github.com/marmotedu/iam/pkg/shutdown"
	"github.com/marmotedu/iam/pkg/shutdown/shutdownmanagers/posixsignal"
	"github.com/marmotedu/iam/pkg/storage"
)

// apiserver 应用配置，包括
// 1. 控制服务优雅启停的功能
// 2. redis配置，apiserver应用使用到了redis
// 3. grpc服务配置，应用需要启动grpc服务
// 4. server服务配置，包括http和https服务，应用需要启动http或https服务
type apiServer struct {
	gs               *shutdown.GracefulShutdown
	redisOptions     *genericoptions.RedisOptions
	gRPCAPIServer    *grpcAPIServer
	genericAPIServer *genericapiserver.GenericAPIServer
}

// 应用启动前的准备工作，在准备函数中可以做各种初始化操作
// 例如初始化数据库、安装业务相关的Gin中间件，安装Restful路由等
type preparedAPIServer struct {
	*apiServer
}

// ExtraConfig defines extra configuration for the iam-apiserver.
// ExtraConfig定义了iam-apiserver服务额外的配置
type ExtraConfig struct {
	Addr         string
	MaxMsgSize   int
	ServerCert   genericoptions.GeneratableKeyCert
	mysqlOptions *genericoptions.MySQLOptions
	// etcdOptions      *genericoptions.EtcdOptions
}

// 构建apiserver实例
func createAPIServer(cfg *config.Config) (*apiServer, error) {
	// 控制优雅关停的服务
	gs := shutdown.New()                                     
	gs.AddShutdownManager(posixsignal.NewPosixSignalManager()) // 添加shutdownmanager

	genericConfig, err := buildGenericConfig(cfg) // 传入应用配置创建HTTP/HTTPS的服务配置
	if err != nil {
		return nil, err
	}

	extraConfig, err := buildExtraConfig(cfg) // 传入应用配置创建GRPC的服务配置
	if err != nil {
		return nil, err
	}
	// 通用api服务
	genericServer, err := genericConfig.Complete().New() // 对HTTP服务配置进行补全，然后New一个REST API SERVER实例
	if err != nil {
		return nil, err
	}
	// grpc服务
	extraServer, err := extraConfig.complete().New() // 对GRPC服务配置进行实例，然后New一个GRPC API SERVER实例
	if err != nil {
		return nil, err
	}

	server := &apiServer{
		gs:               gs,
		redisOptions:     cfg.RedisOptions, // redis配置从应用配置中获取
		genericAPIServer: genericServer,
		gRPCAPIServer:    extraServer,
	}

	return server, nil
}

// PrepareRun 应用的准备工作，包含初始化操作
func (s *apiServer) PrepareRun() preparedAPIServer {
	initRouter(s.genericAPIServer.Engine) // 初始化API路由

	s.initRedisStore() // Redis初始化

	// 添加优雅停止的操作
	s.gs.AddShutdownCallback(shutdown.ShutdownFunc(func(string) error {
		mysqlStore, _ := mysql.GetMySQLFactoryOr(nil)
		if mysqlStore != nil {
			_ = mysqlStore.Close() // 关闭mysql连接池
		}

		s.gRPCAPIServer.Close()    // 关闭grpc服务
		s.genericAPIServer.Close() // 关闭http服务

		return nil
	}))

	// 返回准备好的apiserver实例
	return preparedAPIServer{s}
}

// Run 准备好的apiserver实例执行运行操作
func (s preparedAPIServer) Run() error {
	go s.gRPCAPIServer.Run() // 运行grpc服务

	// start shutdown managers
	if err := s.gs.Start(); err != nil {
		log.Fatalf("start shutdown manager failed: %s", err.Error())
	}

	return s.genericAPIServer.Run()
}

type completedExtraConfig struct {
	*ExtraConfig
}

// Complete fills in any fields not set that are required to have valid data and can be derived from other fields.
// 对GRPC服务配置进行补全
func (c *ExtraConfig) complete() *completedExtraConfig {
	if c.Addr == "" { // 如果grpc服务没有配置监听地址则配置默认地址
		c.Addr = "127.0.0.1:8081"
	}

	return &completedExtraConfig{c}
}

// New create a grpcAPIServer instance.
func (c *completedExtraConfig) New() (*grpcAPIServer, error) {
	// 创建grpc服务
	creds, err := credentials.NewServerTLSFromFile(c.ServerCert.CertKey.CertFile, c.ServerCert.CertKey.KeyFile)
	if err != nil {
		log.Fatalf("Failed to generate credentials %s", err.Error())
	}
	opts := []grpc.ServerOption{grpc.MaxRecvMsgSize(c.MaxMsgSize), grpc.Creds(creds)}
	grpcServer := grpc.NewServer(opts...)

	storeIns, _ := mysql.GetMySQLFactoryOr(c.mysqlOptions) // 根据mysql options创建存储工厂实例
	// storeIns, _ := etcd.GetEtcdFactoryOr(c.etcdOptions, nil)
	store.SetClient(storeIns)
	cacheIns, err := cachev1.GetCacheInsOr(storeIns) // 获取缓存服务
	if err != nil {
		log.Fatalf("Failed to get cache instance: %s", err.Error())
	}

	pb.RegisterCacheServer(grpcServer, cacheIns)

	reflection.Register(grpcServer)

	return &grpcAPIServer{grpcServer, c.Addr}, nil
}

// buildGenericConfig 根据应用配置创建HTTP服务配置
func buildGenericConfig(cfg *config.Config) (genericConfig *genericapiserver.Config, lastErr error) {
	genericConfig = genericapiserver.NewConfig() // 通用的服务配置
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

// nolint: unparam
func buildExtraConfig(cfg *config.Config) (*ExtraConfig, error) {
	return &ExtraConfig{
		Addr:         fmt.Sprintf("%s:%d", cfg.GRPCOptions.BindAddress, cfg.GRPCOptions.BindPort), // 设置grpc服务的监听地址
		MaxMsgSize:   cfg.GRPCOptions.MaxMsgSize,
		ServerCert:   cfg.SecureServing.ServerCert,
		mysqlOptions: cfg.MySQLOptions,
		// etcdOptions:      cfg.EtcdOptions,
	}, nil
}

func (s *apiServer) initRedisStore() {
	ctx, cancel := context.WithCancel(context.Background())
	s.gs.AddShutdownCallback(shutdown.ShutdownFunc(func(string) error {
		cancel()

		return nil
	}))

	config := &storage.Config{
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

	// try to connect to redis
	go storage.ConnectToRedis(ctx, config)
}
