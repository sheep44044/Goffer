package qdrant

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

func (v *VectorStore) Search(ctx context.Context, query string) ([]*schema.Document, error) {
	// 调用 Retriever 的 Retrieve 方法
	// 注意：这里的 TopK 数量是由你在 NewVectorStore 初始化 Retriever 时配置的 TopK: 5 决定的
	docs, err := v.Retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("qdrant 检索失败: %w", err)
	}
	return docs, nil
}
