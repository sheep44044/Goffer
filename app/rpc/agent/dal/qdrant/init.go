package qdrant

import (
	"Goffer/app/rpc/agent/config"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitQdrantClient(cfg *config.Config) *qdrant.Client {
	config := &qdrant.Config{
		Host:   cfg.Qdrant.Host,
		Port:   cfg.Qdrant.Port,
		APIKey: cfg.Qdrant.APIKey,
		UseTLS: false,
	}

	// Mac 本地 Docker 跑 Qdrant 通常没有 TLS，配置非安全传输
	if !config.UseTLS {
		config.GrpcOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
	}

	// 1. 初始化原生 Qdrant 客户端
	client, err := qdrant.NewClient(config)
	if err != nil {
		panic("无法连接 Qdrant 数据库: " + err.Error())
	}

	return client
}
