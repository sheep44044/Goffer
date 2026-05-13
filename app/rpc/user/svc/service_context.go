package svc

import (
	"Goffer/app/rpc/user/config"
	"Goffer/app/rpc/user/dal/cache"
	"Goffer/app/rpc/user/dal/db"
	"Goffer/app/rpc/user/dal/minio"
	"Goffer/app/rpc/user/mq"
	"Goffer/pkg/jwt"

	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config *config.Config
	DB     *db.DBManager
	Cache  *redis.Client
	Minio  *minio.FileStorage
	Kafka  *mq.KafkaProducer
	JWT    *jwt.JWTManager
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

	jwtManager := jwt.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Issuer, rdb)

	return &ServiceContext{
		Config: cfg,
		JWT:    jwtManager,
		DB:     dbManager,
		Cache:  rdb,
		Minio:  minio,
		Kafka:  kafka,
	}
}
