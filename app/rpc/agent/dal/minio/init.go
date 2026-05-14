package minio

import (
	"Goffer/app/rpc/agent/config"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStorage struct {
	client    *minio.Client
	bucket    string
	endpoint  string
	publicURL string
}

// NewFileStorage 初始化保持你的原样即可 (略微修复了错误处理)
func NewFileStorage(cfg *config.Config) (*FileStorage, error) {
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		// 返回具体错误给调用方
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
			// 在生产环境中，建议检查 Policy 设置是否成功
			if err := minioClient.SetBucketPolicy(ctx, bucketName, policy); err != nil {
				log.Printf("Warning: Bucket created but failed to set policy: %v", err)
			} else {
				log.Printf("Bucket %s created and policy set.", bucketName)
			}
		} else {
			log.Printf("Failed to create bucket: %v", err)
		}
	} else if errBucket != nil {
		log.Printf("Warning: check bucket exists failed: %v", errBucket)
	}

	return &FileStorage{
		client:    minioClient,
		bucket:    bucketName,
		endpoint:  cfg.MinIO.Endpoint,
		publicURL: cfg.MinIO.PublicURL,
	}, nil
}
