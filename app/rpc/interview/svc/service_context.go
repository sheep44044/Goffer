package svc

import (
	"Goffer/app/rpc/interview/config"
	"Goffer/app/rpc/interview/dal/ai"
	"Goffer/app/rpc/interview/dal/cache"
	"Goffer/app/rpc/interview/dal/mongodb"
	"Goffer/app/rpc/interview/dal/qdrant"
	"Goffer/app/rpc/interview/dal/repo"
	"Goffer/app/rpc/interview/rpc"
	"Goffer/kitex_gen/user/userservice"
	"context"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config      *config.Config
	Cache       *redis.Client
	Mongo       *mongodb.MongoManager
	VectorStore *qdrant.VectorStore
	AI          *ai.AIService
	Repo        *repo.RepoService
	UserClient  userservice.Client
}

func NewServiceContext(cfg *config.Config) *ServiceContext {
	rdb, err := cache.Init(cfg)
	if err != nil {
		panic(err)
	}

	mongo, err := mongodb.NewMongoManager(cfg)
	if err != nil {
		panic(err)
	}

	ai := ai.NewAIService(cfg)

	embedder, err := ark.NewEmbedder(context.Background(), &ark.EmbeddingConfig{
		APIKey: cfg.VolcEngine.Key,
		Model:  cfg.VolcEngine.EmbedModelID,
	})
	if err != nil {
		panic(err)
	}

	vectorStore := qdrant.NewVectorStore(
		cfg.Qdrant.Host,
		cfg.Qdrant.Port,
		"resume_collection",
		cfg.Qdrant.APIKey,
		embedder,
	)

	repo := repo.NewGetChatService(rdb, mongo, vectorStore)

	userRpcClient := rpc.InitUserClient(cfg)
	return &ServiceContext{
		Config:      cfg,
		Cache:       rdb,
		Mongo:       mongo,
		VectorStore: vectorStore,
		AI:          ai,
		Repo:        repo,
		UserClient:  userRpcClient,
	}
}
