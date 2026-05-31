package jd

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
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

	// 组装完整文本
	content := fmt.Sprintf("【职位】%s\n【公司】%s\n【岗位职责】\n%s\n【任职要求】\n%s",
		task.Title, task.Company, task.Responsibilities, task.Requirements)

	// 文档切片
	chunks, err := splitDocument(ctx, content)
	if err != nil {
		return fmt.Errorf("JD 文档切片失败: %w", err)
	}
	logger.InfoCtx(ctx, "JD 切片完成", zap.Int("chunk_count", len(chunks)), zap.String("jd_id", task.JDID))

	// 为每个切片附加元数据
	for _, chunk := range chunks {
		if chunk.MetaData == nil {
			chunk.MetaData = make(map[string]any)
		}
		chunk.MetaData["type"] = "jd_bank"
		chunk.MetaData["tags"] = task.Tags
		chunk.MetaData["company"] = task.Company
		chunk.MetaData["title"] = task.Title
	}

	ids, err := w.indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("JD 向量入库失败, JDID: %s, err: %w", task.JDID, err)
	}

	logger.InfoCtx(ctx, "JD 向量入库完成", zap.Int("chunk_count", len(ids)), zap.String("jd_id", task.JDID))
	return nil
}

// splitDocument 递归切割长文本，避免单 chunk 过长导致 embedding 稀释
func splitDocument(ctx context.Context, content string) ([]*schema.Document, error) {
	doc := &schema.Document{
		Content: content,
	}
	srcDocs := []*schema.Document{doc}

	recConfig := &recursive.Config{
		ChunkSize:   500,
		OverlapSize: 50,
		Separators:  []string{"\n\n", "\n", "。", "，", " ", ""},
		KeepType:    recursive.KeepTypeNone,
	}

	charSplitter, err := recursive.NewSplitter(ctx, recConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化切分器失败: %w", err)
	}

	return charSplitter.Transform(ctx, srcDocs)
}
