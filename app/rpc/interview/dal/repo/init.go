package repo

import (
	"Goffer/app/rpc/interview/dal/mongodb"

	"github.com/redis/go-redis/v9"
)

type RepoService struct {
	Cache *redis.Client
	Mongo *mongodb.MongoManager
}

func NewGetChatService(Cache *redis.Client, Mongo *mongodb.MongoManager) *RepoService {
	return &RepoService{
		Cache: Cache,
		Mongo: Mongo,
	}
}
