package rpc

import (
	"Goffer/app/rpc/interview/config"
	"Goffer/kitex_gen/agent/agentservice"
	middleware2 "Goffer/pkg/middleware/rpc"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/streamclient" // 🌟 必须引入流式客户端包
	"github.com/cloudwego/kitex/pkg/retry"
	etcd "github.com/kitex-contrib/registry-etcd"
)

func InitAgentClient(cfg *config.Config) (agentservice.Client, agentservice.StreamClient) {
	r, err := etcd.NewEtcdResolver([]string{cfg.Etcd.Address})
	if err != nil {
		panic(err)
	}

	opts := []client.Option{
		client.WithMiddleware(middleware2.CommonMiddleware),
		client.WithInstanceMW(middleware2.ClientMiddleware),
		client.WithMuxConnection(1),
		client.WithFailureRetry(retry.NewFailurePolicy()),
		client.WithResolver(r),
		client.WithRPCTimeout(time.Duration(cfg.RpcClients["agent"].RpcTimeout) * time.Millisecond),
		client.WithConnectTimeout(time.Duration(cfg.RpcClients["agent"].ConnTimeout) * time.Millisecond),
	}

	c, err := agentservice.NewClient(cfg.RpcClients["agent"].Name, opts...)
	if err != nil {
		panic(err)
	}

	// 2. 流式客户端配置 (专门用于 ChatStream)
	streamOpts := []streamclient.Option{
		streamclient.WithResolver(r), // 必须告诉流式客户端怎么找 Agent 服务
		streamclient.WithConnectTimeout(time.Duration(cfg.RpcClients["agent"].ConnTimeout) * time.Millisecond),
	}

	sc, err := agentservice.NewStreamClient(cfg.RpcClients["agent"].Name, streamOpts...)
	if err != nil {
		panic(err)
	}

	return c, sc
}
