// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/marmotedu/component-base/pkg/core"
	"github.com/marmotedu/component-base/pkg/version"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/marmotedu/iam/internal/pkg/middleware"
	"github.com/marmotedu/iam/pkg/log"
)

// GenericAPIServer contains state for an iam api server.
// type GenericAPIServer gin.Engine.
// GenericAPIServer 包含iam api server的状态
type GenericAPIServer struct {
	middlewares []string // 需要加载的中间件
	// SecureServingInfo holds configuration of the TLS server.
	SecureServingInfo *SecureServingInfo // http服务需要枪的配置

	// InsecureServingInfo holds configuration of the insecure HTTP server.
	InsecureServingInfo *InsecureServingInfo

	// ShutdownTimeout is the timeout used for server shutdown. This specifies the timeout before server
	// gracefully shutdown returns.
	ShutdownTimeout time.Duration

	*gin.Engine
	healthz         bool
	enableMetrics   bool
	enableProfiling bool
	// wrapper for gin.Engine

	insecureServer, secureServer *http.Server
}

func initGenericAPIServer(s *GenericAPIServer) {
	// do some setup
	// s.GET(path, ginSwagger.WrapHandler(swaggerFiles.Handler))

	s.Setup()              // 设置debug日志的输出格式
	s.InstallMiddlewares() // 安装gin中间件
	s.InstallAPIs()        // 安装通用的api，包括健康检查、启用Metric、启用pprof
}

// InstallAPIs install generic apis.
// InstallAPIs 安装通用的api
func (s *GenericAPIServer) InstallAPIs() {
	// install healthz handler
	// 安装健康检查处理器
	if s.healthz {
		s.GET("/healthz", func(c *gin.Context) {
			core.WriteResponse(c, nil, map[string]string{"status": "ok"})
		})
	}

	// install metric handler
	// 导出gin相关的metric，以"gin"开头
	if s.enableMetrics {
		prometheus := ginprometheus.NewPrometheus("gin")
		prometheus.Use(s.Engine)
	}

	// install pprof handler
	// 启用性能分析功能
	if s.enableProfiling {
		pprof.Register(s.Engine)
	}

	// 添加版本处理器
	s.GET("/version", func(c *gin.Context) {
		core.WriteResponse(c, nil, version.Get())
	})
}

// Setup do some setup work for gin engine.
func (s *GenericAPIServer) Setup() {
	// 设置gin的debug日志输出格式
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		log.Infof("%-6s %-s --> %s (%d handlers)", httpMethod, absolutePath, handlerName, nuHandlers)
	}
}

// InstallMiddlewares install generic middlewares.
// InstallMiddlewares 安装通用的中间件
func (s *GenericAPIServer) InstallMiddlewares() {
	// necessary middlewares 安装两个必要的中间件
	s.Use(middleware.RequestID()) // 设置请求id
	s.Use(middleware.Context())   // 上下文中添加必要的键值对

	// install custom middlewares 安装自定义的中间件，配置文件中可以设置
	for _, m := range s.middlewares { // 从中间件注册器中寻找配置的中间件
		mw, ok := middleware.Middlewares[m]
		if !ok {
			log.Warnf("can not find middleware: %s", m)

			continue
		}

		log.Infof("install middleware: %s", m)
		s.Use(mw)
	}
}

/*
// preparedGenericAPIServer is a private wrapper that enforces a call of PrepareRun() before Run can be invoked.
type preparedGenericAPIServer struct {
	*GenericAPIServer
}

func (s *GenericAPIServer) PrepareRun() preparedGenericAPIServer {
	return preparedGenericAPIServer{s}
}
*/

// Run spawns the http server. It only returns when the port cannot be listened on initially.
// 运行http服务。它只有在端口无法被监听时返回。
func (s *GenericAPIServer) Run() error {
	// For scalability, use custom HTTP configuration mode here
	s.insecureServer = &http.Server{
		Addr:    s.InsecureServingInfo.Address,
		Handler: s,
		// ReadTimeout:    10 * time.Second,
		// WriteTimeout:   10 * time.Second,
		// MaxHeaderBytes: 1 << 20,

	}

	// For scalability, use custom HTTP configuration mode here
	s.secureServer = &http.Server{
		Addr:    s.SecureServingInfo.Address(),
		Handler: s,
		// ReadTimeout:    10 * time.Second,
		// WriteTimeout:   10 * time.Second,
		// MaxHeaderBytes: 1 << 20,
	}

	var eg errgroup.Group

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	eg.Go(func() error {
		log.Infof("Start to listening the incoming requests on http address: %s", s.InsecureServingInfo.Address)

		if err := s.insecureServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err.Error())

			return err
		}

		log.Infof("Server on %s stopped", s.InsecureServingInfo.Address)

		return nil
	})

	eg.Go(func() error {
		key, cert := s.SecureServingInfo.CertKey.KeyFile, s.SecureServingInfo.CertKey.CertFile
		if cert == "" || key == "" || s.SecureServingInfo.BindPort == 0 {
			return nil
		}

		log.Infof("Start to listening the incoming requests on https address: %s", s.SecureServingInfo.Address())

		if err := s.secureServer.ListenAndServeTLS(cert, key); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err.Error())

			return err
		}

		log.Infof("Server on %s stopped", s.SecureServingInfo.Address())

		return nil
	})

	// Ping the server to make sure the router is working.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if s.healthz {
		if err := s.ping(ctx); err != nil {
			return err
		}
	}

	if err := eg.Wait(); err != nil {
		log.Fatal(err.Error())
	}

	return nil
}

// Close graceful shutdown the api server.
// 优雅关闭api server服务
func (s *GenericAPIServer) Close() {
	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.secureServer.Shutdown(ctx); err != nil {
		log.Warnf("Shutdown secure server failed: %s", err.Error())
	}

	if err := s.insecureServer.Shutdown(ctx); err != nil {
		log.Warnf("Shutdown insecure server failed: %s", err.Error())
	}
}

// ping pings the http server to make sure the router is working.
// 健康检查插件
func (s *GenericAPIServer) ping(ctx context.Context) error {
	url := fmt.Sprintf("http://%s/healthz", s.InsecureServingInfo.Address)
	if strings.Contains(s.InsecureServingInfo.Address, "0.0.0.0") {
		url = fmt.Sprintf("http://127.0.0.1:%s/healthz", strings.Split(s.InsecureServingInfo.Address, ":")[1])
	}

	for {
		// Change NewRequest to NewRequestWithContext and pass context it
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		// Ping the server by sending a GET request to `/healthz`.

		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Info("The router has been deployed successfully.")

			resp.Body.Close()

			return nil
		}

		// Sleep for a second to continue the next ping.
		log.Info("Waiting for the router, retry in 1 second.")
		time.Sleep(1 * time.Second)

		select {
		case <-ctx.Done():
			log.Fatal("can not ping http server within the specified time interval.")
		default:
		}
	}
	// return fmt.Errorf("the router has no response, or it might took too long to start up")
}
