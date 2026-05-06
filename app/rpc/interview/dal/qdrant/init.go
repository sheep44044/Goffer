package qdrant

import (
	"context"

	qdrant_idx "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	qdrant_ret "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type VectorStore struct {
	Indexer   indexer.Indexer
	Retriever retriever.Retriever
	RawClient *qdrant.Client // 保留原生客户端，以防不时之需（比如你想用原生 API 删库）
}

// NewVectorStore 负责初始化原生客户端和官方组件
func NewVectorStore(host string, port int, collection string, apiKey string, embedder embedding.Embedder) *VectorStore {
	config := &qdrant.Config{
		Host: host,
		Port: port,
	}

	if apiKey != "" {
		config.APIKey = apiKey
		config.UseTLS = false // 本地或内网开发通常设为 false
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

	ctx := context.Background()

	// 2. 初始化官方 Indexer (使用正确的包路径 qdrant_idx)
	idx, err := qdrant_idx.NewIndexer(ctx, &qdrant_idx.Config{
		Client:     client,
		Collection: collection,
		VectorDim:  2048, // 替换为你使用的 Embedding 模型的实际维度 (如 OpenAI 的 text-embedding-3-small 是 1536)
		Distance:   qdrant.Distance_Cosine,
		BatchSize:  10,
		Embedding:  embedder,
	})
	if err != nil {
		panic("初始化 Qdrant Indexer 失败: " + err.Error())
	}

	// 3. 初始化官方 Retriever (使用正确的包路径 qdrant_ret)
	ret, err := qdrant_ret.NewRetriever(ctx, &qdrant_ret.Config{
		Client:     client,
		Collection: collection,
		Embedding:  embedder,
		TopK:       5,
	})
	if err != nil {
		panic("初始化 Qdrant Retriever 失败: " + err.Error())
	}

	return &VectorStore{
		Indexer:   idx,
		Retriever: ret,
		RawClient: client,
	}
}
