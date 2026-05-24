package minio

import (
	"Goffer/app/rpc/agent/config"
	"Goffer/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type FileStorage struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	publicURL string
}

func NewFileStorage(cfg *config.Config) (*FileStorage, error) {
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bucketName := cfg.MinIO.Bucket
	exists, errBucket := minioClient.BucketExists(ctx, bucketName)
	if errBucket == nil && !exists {
		err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err == nil {
			policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucketName)
			if err := minioClient.SetBucketPolicy(ctx, bucketName, policy); err != nil {
				logger.Warn("Bucket 已创建但设置策略失败", zap.String("bucket", bucketName), zap.Error(err))
			} else {
				logger.Info("Bucket 已创建并设置策略", zap.String("bucket", bucketName))
			}
		} else {
			logger.Warn("创建 Bucket 失败", zap.String("bucket", bucketName), zap.Error(err))
		}
	} else if errBucket != nil {
		logger.Warn("检查 Bucket 是否存在失败", zap.String("bucket", bucketName), zap.Error(errBucket))
	}

	return &FileStorage{
		client:    minioClient,
		bucket:    bucketName,
		endpoint:  cfg.MinIO.Endpoint,
		publicURL: cfg.MinIO.PublicURL,
	}, nil
}
