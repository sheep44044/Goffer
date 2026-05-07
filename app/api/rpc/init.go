package rpc

import (
	"Goffer/app/api/config"
	"Goffer/kitex_gen/interview/interviewservice"
	"Goffer/kitex_gen/user/userservice"
	middleware2 "Goffer/pkg/middleware/rpc"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/streamclient"
	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/retry"
	etcd "github.com/kitex-contrib/registry-etcd"
)

// InitRpcClients 一次性初始化所有的下游 RPC 客户端
func InitRpcClients(cfg *config.Config) {
	// 1. 初始化统一的 Etcd 解析器 (大家都去同一个通讯录找人)
	r, err := etcd.NewEtcdResolver([]string{cfg.Etcd.Address})
	if err != nil {
		panic(err)
	}

	// 2. 提取公共的配置项 (像公用的中间件、复用连接等)
	commonOptions := []client.Option{
		client.WithMiddleware(middleware2.CommonMiddleware),
		client.WithInstanceMW(middleware2.ClientMiddleware),
		client.WithMuxConnection(1),
		client.WithFailureRetry(retry.NewFailurePolicy()),
		// client.WithSuite(trace.NewDefaultClientSuite()),   // tracer
		client.WithResolver(r),
	}

	// 3. 初始化 User RPC 客户端
	initUserClient(cfg.RpcClients["user"], commonOptions)

	// 4. 初始化 Interview RPC 客户端
	initInterviewClient(cfg.RpcClients["interview"], commonOptions, r)
}

func initInterviewClient(cfg config.RpcClientConfig, commonOpts []client.Option, r discovery.Resolver) {
	opts := append(commonOpts,
		client.WithRPCTimeout(time.Duration(cfg.RpcTimeout)*time.Millisecond),
		client.WithConnectTimeout(time.Duration(cfg.ConnTimeout)*time.Millisecond),
	)

	// 1. 初始化普通客户端 (用于常规 Unary 调用)
	c, err := interviewservice.NewClient(cfg.Name, opts...)
	if err != nil {
		panic(err)
	}
	interviewClient = c

	streamOpts := []streamclient.Option{
		streamclient.WithResolver(r), // 告诉流式客户端去哪里找服务
		streamclient.WithConnectTimeout(time.Duration(cfg.ConnTimeout) * time.Millisecond),
	}

	sc, err := interviewservice.NewStreamClient(cfg.Name, streamOpts...)
	if err != nil {
		panic(err)
	}
	interviewStreamClient = sc
}

func initUserClient(cfg config.RpcClientConfig, commonOpts []client.Option) {
	opts := append(commonOpts,
		client.WithRPCTimeout(time.Duration(cfg.RpcTimeout)*time.Millisecond),
		client.WithConnectTimeout(time.Duration(cfg.ConnTimeout)*time.Millisecond),
	)

	c, err := userservice.NewClient(cfg.Name, opts...)
	if err != nil {
		panic(err)
	}
	userClient = c
}

func UserClient() userservice.Client {
	return userClient
}

func InterviewClient() interviewservice.Client {
	return interviewClient
}
