package interview

import (
	"Goffer/app/rpc/interview/config"
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/interview/interviewservice"
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

func Init(cfg *config.Config) {
	svc.NewServiceContext(cfg)
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

	Init(cfg)

	svr := interviewservice.NewServer(new(interviewServiceImpl),
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
