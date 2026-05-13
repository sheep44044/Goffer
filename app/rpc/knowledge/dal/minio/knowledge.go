package minio

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// UploadFile 泛化的文件上传方法 (适配 Kitex RPC 的 []byte)
func (s *FileStorage) UploadFile(ctx context.Context, originalName string, fileData []byte, contentType string) (string, string, error) {
	// 1. 生成基于 UUID 的安全文件名，防止覆盖
	// 提取原文件的后缀名 (例如: .pdf, .png)
	ext := filepath.Ext(originalName)
	// 生成新文件名: 类似 550e8400-e29b-41d4-a716-446655440000.pdf
	safeFileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// 2. 将 []byte 转换为 io.Reader
	reader := bytes.NewReader(fileData)
	fileSize := int64(len(fileData))

	// 3. 上传到 MinIO
	_, err := s.client.PutObject(ctx, s.bucket, safeFileName, reader, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", "", err
	}

	// 4. 拼接并返回 URL
	baseURL := strings.TrimRight(s.publicURL, "/")
	fileURL := fmt.Sprintf("%s/%s/%s", baseURL, s.bucket, safeFileName)

	// 返回新文件名和 URL，新文件名后续可能要存入数据库
	return safeFileName, fileURL, nil
}
