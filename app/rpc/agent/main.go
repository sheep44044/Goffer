package main

import (
	"Goffer/app/rpc/agent/bot"
	"Goffer/app/rpc/agent/config"
	"Goffer/app/rpc/agent/svc"
	"Goffer/app/rpc/agent/tools"
	"Goffer/app/rpc/agent/worker"
	"Goffer/kitex_gen/agent/agentservice"
	"Goffer/pkg/contextutil"
	"Goffer/pkg/logger"
	middleware2 "Goffer/pkg/middleware/rpc"
	"context"
	"fmt"
	"net"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/limit"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"go.uber.org/zap"
)

func Init(cfg *config.Config) *svc.ServiceContext {
	return svc.NewServiceContext(cfg)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("配置加载失败: %w", err))
	}
	r, err := etcd.NewEtcdRegistry([]string{cfg.Etcd.Address})
	if err != nil {
		panic(fmt.Errorf("连接 Etcd 失败: %w", err))
	}

	ip, err := contextutil.GetOutBoundIP()
	if err != nil {
		panic(err)
	}

	listenAddr := fmt.Sprintf("%s:%s", ip, cfg.Server.Port)
	addr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		panic(err)
	}

	svc := Init(cfg)
	botManager := bot.InitBotManager(svc)
	if err := tools.InitTools(); err != nil {
		panic(fmt.Errorf("初始化 MCP 工具失败: %w", err))
	}
	botManager.LoadAllPresets()

	// 启动 Redis Pub/Sub 打断事件订阅
	svc.CancelManager.SubscribeCancelEvents(svc.RedisClient)

	mqEngine := worker.NewMQEngine(svc)
	go mqEngine.Start(context.Background())

	svr := agentservice.NewServer(NewAgentService(svc),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: cfg.Service.Name}),
		server.WithMiddleware(middleware2.CommonMiddleware), // middleware
		server.WithMiddleware(middleware2.ServerMiddleware),
		server.WithServiceAddr(addr),                                       // address
		server.WithLimit(&limit.Option{MaxConnections: 1000, MaxQPS: 100}), // limit
		server.WithMuxTransport(),                                          // Multiplex
		//server.WithSuite(trace.NewDefaultServerSuite()),                    // tracer
		//server.WithBoundHandler(bound.NewCpuLimitHandler()), // BoundHandler
		server.WithSuite(tracing.NewServerSuite()),
		server.WithRegistry(r), // registry
	)
	logger.Info("Agent RPC Server 正在启动", zap.String("listen_addr", listenAddr), zap.String("etcd", cfg.Etcd.Address))

	err = svr.Run()
	if err != nil {
		klog.Fatal(err)
	}
}
