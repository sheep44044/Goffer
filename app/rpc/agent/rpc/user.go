package rpc

import (
	"Goffer/app/rpc/agent/config"
	"Goffer/kitex_gen/user/userservice"
	middleware2 "Goffer/pkg/middleware/rpc"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/retry"
	etcd "github.com/kitex-contrib/registry-etcd"
)

func InitUserClient(cfg *config.Config) userservice.Client {
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
		// 读取 Interview 配置里的 UserClient 超时设置
		client.WithRPCTimeout(time.Duration(cfg.RpcClients["user"].RpcTimeout) * time.Millisecond),
		client.WithConnectTimeout(time.Duration(cfg.RpcClients["user"].ConnTimeout) * time.Millisecond),
	}

	c, err := userservice.NewClient(cfg.RpcClients["user"].Name, opts...)
	if err != nil {
		panic(err)
	}

	return c // 直接返回客户端实例
}
