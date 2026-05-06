package svc

import (
	"Goffer/app/rpc/user/config"
	"Goffer/app/rpc/user/dal/ai"
	"Goffer/app/rpc/user/dal/cache"
	"Goffer/app/rpc/user/dal/db"
	"Goffer/app/rpc/user/dal/minio"
	"Goffer/app/rpc/user/dal/qdrant"
	"Goffer/app/rpc/user/mq"
	"Goffer/pkg/util"
	"context"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config      *config.Config
	DB          *db.DBManager
	Cache       *redis.Client
	Minio       *minio.FileStorage
	Kafka       *mq.KafkaProducer
	AI          *ai.AIService
	JWT         *util.JWTManager
	VectorStore *qdrant.VectorStore
}

func NewServiceContext(cfg *config.Config) *ServiceContext {
	dbConn, err := db.Init(cfg)
	if err != nil {
		panic(err)
	}
	dbManager := db.NewDBManager(dbConn)

	rdb, err := cache.Init(cfg)
	if err != nil {
		panic(err)
	}

	minio, err := minio.NewFileStorage(cfg)
	if err != nil {
		panic(err)
	}

	kafka := mq.InitProducer(cfg)

	ai := ai.NewAIService(cfg)

	jwtManager := util.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Issuer, rdb)

	embedder, err := ark.NewEmbedder(context.Background(), &ark.EmbeddingConfig{
		APIKey: cfg.VolcEngine.Key,
		Model:  cfg.VolcEngine.EmbedModelID,
	})
	if err != nil {
		panic(err)
	}

	// 2. 将 embedder 注入给 Qdrant 包装器
	vectorStore := qdrant.NewVectorStore(
		cfg.Qdrant.Host,
		cfg.Qdrant.Port,
		"resume_collection",
		cfg.Qdrant.APIKey,
		embedder,
	)

	return &ServiceContext{
		Config:      cfg,
		DB:          dbManager,
		Cache:       rdb,
		Minio:       minio,
		Kafka:       kafka,
		AI:          ai,
		JWT:         jwtManager,
		VectorStore: vectorStore,
	}
}
