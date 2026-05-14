package question

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/indexer/qdrant"
	"github.com/cloudwego/eino/schema"
)

type QuestionParseTask struct {
	QuestionID      string   `json:"question_id"`
	QuestionContent string   `json:"question_content"`
	StandardAnswer  string   `json:"standard_answer"`
	Tags            []string `json:"tags"`
	Difficulty      string   `json:"difficulty"`
}

func (w *QuestionWorker) IngestQuestion(ctx context.Context, jsonData []byte) error {
	fmt.Println("[RAG 服务] 开始处理单道题目入库任务...")

	// 1. 解析 JSON 数据 (与 Service 层投递的数据结构保持绝对一致)
	var task QuestionParseTask
	if err := json.Unmarshal(jsonData, &task); err != nil {
		return fmt.Errorf("解析 Kafka 题目数据失败: %w", err)
	}

	// 2. 将结构化题目直接转换为 Eino Documents
	// 将问题和标准答案拼接成供大模型阅读的 Context
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

	// Eino 的接口要求传入数组，我们把单条 doc 包进数组即可
	chunks := []*schema.Document{doc}

	fmt.Printf("-> 1. 题目解析完成，QuestionID: %s\n", task.QuestionID)

	// 3. 初始化针对题库 Collection 的 Qdrant Indexer
	indexer, err := qdrant.NewIndexer(ctx, &qdrant.Config{
		Client:     w.svc.QdrantClient,
		Collection: "goffer_question_bank",
	})
	if err != nil {
		return fmt.Errorf("初始化 Qdrant Indexer 失败: %w", err)
	}

	// 4. 执行向量化并存入 Qdrant
	// 这里 Eino 框架内部会自动调用 Embedding 模型将 Content 转化为向量，并连同 Metadata 一起存入
	ids, err := indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("题目向量入库失败, QuestionID: %s, err: %w", task.QuestionID, err)
	}

	fmt.Printf("-> 2. 题目向量入库完成，成功存入 Qdrant, ID: %s！\n", ids[0])
	return nil
}
