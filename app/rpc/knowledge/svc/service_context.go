package svc

import (
	"Goffer/app/rpc/knowledge/config"
	"Goffer/app/rpc/knowledge/dal/ai"
	"Goffer/app/rpc/knowledge/dal/db"
	"Goffer/app/rpc/knowledge/dal/minio"
	"Goffer/app/rpc/knowledge/mq"
)

type ServiceContext struct {
	Config *config.Config
	DB     *db.DBManager
	Kafka  *mq.KafkaProducer
	AI     *ai.AIService
	Minio  *minio.FileStorage
}

func NewServiceContext(cfg *config.Config) *ServiceContext {
	dbConn, err := db.Init(cfg)
	if err != nil {
		panic(err)
	}
	dbManager := db.NewDBManager(dbConn)

	kafka := mq.InitProducer(cfg)

	ai := ai.NewAIService(cfg)

	minio, err := minio.NewFileStorage(cfg)
	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config: cfg,
		DB:     dbManager,
		Kafka:  kafka,
		AI:     ai,
		Minio:  minio,
	}
}
