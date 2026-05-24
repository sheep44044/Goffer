package question

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

	content := fmt.Sprintf("【面试题】%s\n【标准答案】%s", task.QuestionContent, task.StandardAnswer)

	doc := &schema.Document{
		ID:      task.QuestionID,
		Content: content,
		MetaData: map[string]any{
			"type":       "question_bank",
			"tags":       task.Tags,
			"difficulty": task.Difficulty,
		},
	}
	chunks := []*schema.Document{doc}
	logger.InfoCtx(ctx, "题目解析完成", zap.String("question_id", task.QuestionID))

	ids, err := w.indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("题目向量入库失败, QuestionID: %s, err: %w", task.QuestionID, err)
	}

	logger.InfoCtx(ctx, "题目向量入库完成", zap.String("qdrant_id", ids[0]))
	return nil
}
