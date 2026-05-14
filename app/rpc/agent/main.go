package agent

import (
	"Goffer/app/rpc/agent/config"
	"Goffer/app/rpc/agent/svc"
	"Goffer/app/rpc/agent/worker"
	"Goffer/kitex_gen/agent/agentservice"
	"Goffer/pkg/contextutil"
	middleware2 "Goffer/pkg/middleware/rpc"
	"fmt"
	"net"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/limit"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
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

	mqEngine := worker.NewMQEngine(svc)
	go mqEngine.Start()

	svr := agentservice.NewServer(NewAgentService(svc),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: cfg.Service.Name}),
		server.WithMiddleware(middleware2.CommonMiddleware), // middleware
		server.WithMiddleware(middleware2.ServerMiddleware),
		server.WithServiceAddr(addr),                                       // address
		server.WithLimit(&limit.Option{MaxConnections: 1000, MaxQPS: 100}), // limit
		server.WithMuxTransport(),                                          // Multiplex
		//server.WithSuite(trace.NewDefaultServerSuite()),                    // tracer
		//server.WithBoundHandler(bound.NewCpuLimitHandler()), // BoundHandler
		server.WithRegistry(r), // registry
	)
	fmt.Printf("User RPC Server 正在启动，监听地址: %s, 注册到 Etcd: %s\n", listenAddr, cfg.Etcd.Address)

	err = svr.Run()
	if err != nil {
		klog.Fatal(err)
	}
}
