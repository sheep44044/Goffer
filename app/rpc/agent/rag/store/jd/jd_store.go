package jd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/indexer/qdrant"
	"github.com/cloudwego/eino/schema"
)

type JDParseTask struct {
	JDID             string   `json:"jd_id"`
	Company          string   `json:"company"`
	Title            string   `json:"title"`
	Responsibilities string   `json:"responsibilities"`
	Requirements     string   `json:"requirements"`
	Tags             []string `json:"tags"`
}

func (w *JDWorker) IngestJD(ctx context.Context, jsonData []byte) error {
	fmt.Println("[RAG 服务] 开始处理单条 JD 入库任务...")

	// 1. 解析 JSON 数据
	var task JDParseTask
	if err := json.Unmarshal(jsonData, &task); err != nil {
		return fmt.Errorf("解析 Kafka JD 数据失败: %w", err)
	}

	// 2. 将结构化题目直接转换为 Eino Documents
	// 🌟 重新拼接专属于 JD 的 Context，让大模型在检索时能准确理解
	content := fmt.Sprintf("【职位】%s\n【公司】%s\n【岗位职责】\n%s\n【任职要求】\n%s",
		task.Title, task.Company, task.Responsibilities, task.Requirements)

	doc := &schema.Document{
		ID:      task.JDID,
		Content: content,
		MetaData: map[string]any{
			"type":    "jd_bank",
			"tags":    task.Tags,
			"company": task.Company, // 将公司和岗位名称作为元数据，方便后期精确过滤
			"title":   task.Title,
		},
	}

	chunks := []*schema.Document{doc}

	fmt.Printf("-> 1. JD 解析完成，JDID: %s\n", task.JDID)

	// 3. 初始化针对 JD Collection 的 Qdrant Indexer
	indexer, err := qdrant.NewIndexer(ctx, &qdrant.Config{
		Client:     w.svc.QdrantClient,
		Collection: "goffer_jd_bank", // 🌟 使用专属的 JD 向量集合
	})
	if err != nil {
		return fmt.Errorf("初始化 Qdrant Indexer 失败: %w", err)
	}

	// 4. 执行向量化并存入 Qdrant
	ids, err := indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("JD 向量入库失败, JDID: %s, err: %w", task.JDID, err)
	}

	fmt.Printf("-> 2. JD 向量入库完成，成功存入 Qdrant, ID: %s！\n", ids[0])
	return nil
}
