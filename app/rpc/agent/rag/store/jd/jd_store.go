package jd

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"

	qdrant_indexer "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

type JDWorker struct {
	svc     *svc.ServiceContext
	indexer indexer.Indexer
}

func NewJDWorker(svc *svc.ServiceContext) (*JDWorker, error) {
	ctx := context.Background()

	idx, err := qdrant_indexer.NewIndexer(ctx, &qdrant_indexer.Config{
		Client:     svc.QdrantClient,
		Collection: "goffer_jd_bank",
		Embedding:  svc.Embedder,
		VectorDim:  2048,
		Distance:   qdrant.Distance_Cosine,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 JD Indexer 失败: %w", err)
	}

	return &JDWorker{
		svc:     svc,
		indexer: idx,
	}, nil
}

type JDParseTask struct {
	JDID             string   `json:"jd_id"`
	Company          string   `json:"company"`
	Title            string   `json:"title"`
	Responsibilities string   `json:"responsibilities"`
	Requirements     string   `json:"requirements"`
	Tags             []string `json:"tags"`
}

func (w *JDWorker) IngestJD(ctx context.Context, jsonData []byte) error {
	logger.InfoCtx(ctx, "开始处理单条 JD 入库任务")

	var task JDParseTask
	if err := json.Unmarshal(jsonData, &task); err != nil {
		return fmt.Errorf("解析 Kafka JD 数据失败: %w", err)
	}

	content := fmt.Sprintf("【职位】%s\n【公司】%s\n【岗位职责】\n%s\n【任职要求】\n%s",
		task.Title, task.Company, task.Responsibilities, task.Requirements)

	doc := &schema.Document{
		ID:      task.JDID,
		Content: content,
		MetaData: map[string]any{
			"type":    "jd_bank",
			"tags":    task.Tags,
			"company": task.Company,
			"title":   task.Title,
		},
	}
	chunks := []*schema.Document{doc}
	logger.InfoCtx(ctx, "JD 解析完成", zap.String("jd_id", task.JDID))

	ids, err := w.indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("JD 向量入库失败, JDID: %s, err: %w", task.JDID, err)
	}

	logger.InfoCtx(ctx, "JD 向量入库完成", zap.String("qdrant_id", ids[0]))
	return nil
}
