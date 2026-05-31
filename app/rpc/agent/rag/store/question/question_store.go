package question

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

type QuestionWorker struct {
	svc     *svc.ServiceContext
	indexer indexer.Indexer
}

func NewQuestionWorker(svc *svc.ServiceContext) (*QuestionWorker, error) {
	ctx := context.Background()

	idx, err := qdrant_indexer.NewIndexer(ctx, &qdrant_indexer.Config{
		Client:     svc.QdrantClient,
		Collection: "goffer_question_bank",
		Embedding:  svc.Embedder,
		VectorDim:  2048,
		Distance:   qdrant.Distance_Cosine,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 Question Indexer 失败: %w", err)
	}

	return &QuestionWorker{
		svc:     svc,
		indexer: idx,
	}, nil
}

type QuestionParseTask struct {
	QuestionID      string   `json:"question_id"`
	QuestionContent string   `json:"question_content"`
	StandardAnswer  string   `json:"standard_answer"`
	Tags            []string `json:"tags"`
	Difficulty      string   `json:"difficulty"`
}

func (w *QuestionWorker) IngestQuestion(ctx context.Context, jsonData []byte) error {
	logger.InfoCtx(ctx, "开始处理单道题目入库任务")

	var task QuestionParseTask
	if err := json.Unmarshal(jsonData, &task); err != nil {
		return fmt.Errorf("解析 Kafka 题目数据失败: %w", err)
	}

	// 组装题目与标准答案
	content := fmt.Sprintf("【面试题】%s\n【标准答案】%s", task.QuestionContent, task.StandardAnswer)

	// 文档切片（标准答案可能很长，递归切割提升 embedding 质量）
	chunks, err := splitDocument(ctx, content)
	if err != nil {
		return fmt.Errorf("题目文档切片失败: %w", err)
	}
	logger.InfoCtx(ctx, "题目切片完成", zap.Int("chunk_count", len(chunks)), zap.String("question_id", task.QuestionID))

	// 为每个切片附加元数据
	for _, chunk := range chunks {
		if chunk.MetaData == nil {
			chunk.MetaData = make(map[string]any)
		}
		chunk.MetaData["type"] = "question_bank"
		chunk.MetaData["tags"] = task.Tags
		chunk.MetaData["difficulty"] = task.Difficulty
	}

	ids, err := w.indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("题目向量入库失败, QuestionID: %s, err: %w", task.QuestionID, err)
	}

	logger.InfoCtx(ctx, "题目向量入库完成", zap.Int("chunk_count", len(ids)), zap.String("question_id", task.QuestionID))
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
