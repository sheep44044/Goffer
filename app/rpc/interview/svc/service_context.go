package svc

import (
	"Goffer/app/rpc/interview/config"
	"Goffer/app/rpc/interview/dal/cache"
	"Goffer/app/rpc/interview/dal/mongodb"
	"Goffer/app/rpc/interview/dal/repo"
	"Goffer/app/rpc/interview/rpc"
	"Goffer/kitex_gen/agent/agentservice"
	"Goffer/kitex_gen/user/userservice"

	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config            *config.Config
	Cache             *redis.Client
	Mongo             *mongodb.MongoManager
	Repo              *repo.RepoService
	UserClient        userservice.Client
	AgentClient       agentservice.Client
	AgentStreamClient agentservice.StreamClient
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

	repo := repo.NewGetChatService(rdb, mongo)

	userRpcClient := rpc.InitUserClient(cfg)
	agentRpcClient, agentStream := rpc.InitAgentClient(cfg)

	return &ServiceContext{
		Config:            cfg,
		Cache:             rdb,
		Mongo:             mongo,
		Repo:              repo,
		UserClient:        userRpcClient,
		AgentClient:       agentRpcClient,
		AgentStreamClient: agentStream,
	}
}
