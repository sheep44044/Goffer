package mongodb

import (
	"Goffer/app/rpc/interview/config"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoManager struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func NewMongoManager(cfg *config.Config) (*MongoManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MongoDB.Timeout)*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.MongoDB.URI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	fmt.Println("MongoDB initialized successfully (DI Mode)!")

	return &MongoManager{
		Client: client,
		DB:     client.Database(cfg.MongoDB.Database),
	}, nil
}

// Close 优雅释放连接
func (m *MongoManager) Close() {
	if m.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.Client.Disconnect(ctx); err != nil {
			fmt.Printf("Error disconnecting MongoDB: %v\n", err)
		}
	}
}
