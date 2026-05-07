package repo

import (
	"Goffer/app/rpc/interview/dal/mongodb"
	"Goffer/app/rpc/interview/dal/qdrant"

	"github.com/redis/go-redis/v9"
)

type RepoService struct {
	Cache       *redis.Client
	Mongo       *mongodb.MongoManager
	VectorStore *qdrant.VectorStore
}

func NewGetChatService(Cache *redis.Client, Mongo *mongodb.MongoManager, VectorStore *qdrant.VectorStore) *RepoService {
	return &RepoService{
		Cache:       Cache,
		Mongo:       Mongo,
		VectorStore: VectorStore,
	}
}
