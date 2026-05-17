package svc

import (
	"Goffer/app/rpc/agent/config"
	"Goffer/app/rpc/agent/dal/ai"
	"Goffer/app/rpc/agent/dal/minio"
	custom_qdrant "Goffer/app/rpc/agent/dal/qdrant"
	"Goffer/app/rpc/agent/internal/cancelmgr"
	"Goffer/app/rpc/agent/rpc"
	"Goffer/kitex_gen/interview/interviewservice"
	"Goffer/kitex_gen/user/userservice"
	"context"
	"fmt"

	ark_model "github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config          *config.Config
	Minio           *minio.FileStorage
	QdrantClient    *qdrant.Client
	AI              *ai.AIService
	UserClient      userservice.Client
	InterviewClient interviewservice.Client
	EinoChatModel   model.ToolCallingChatModel
	RedisClient     *redis.Client
	CancelManager   *cancelmgr.CancelManager // 全链路打断管理
}

func NewServiceContext(cfg *config.Config) *ServiceContext {
	userRpcClient := rpc.InitUserClient(cfg)

	minio, err := minio.NewFileStorage(cfg)
	if err != nil {
		panic(err)
	}

	qdrant := custom_qdrant.InitQdrantClient(cfg)

	ai := ai.NewAIService(cfg)

	chatModel, err := ark_model.NewChatModel(context.Background(), &ark_model.ChatModelConfig{
		APIKey: cfg.VolcEngine.Key,
		Model:  cfg.VolcEngine.ChatModelID,
	})
	if err != nil {
		panic(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	cancelMgr := cancelmgr.NewCancelManager()

	return &ServiceContext{
		Config:        cfg,
		Minio:         minio,
		QdrantClient:  qdrant,
		UserClient:    userRpcClient,
		AI:            ai,
		EinoChatModel: chatModel,
		RedisClient:   redisClient,
		CancelManager: cancelMgr,
	}
}
