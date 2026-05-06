package worker

import (
	"Goffer/app/rpc/user/mq"
	"Goffer/app/rpc/user/svc"
	"Goffer/pkg/pdfparser"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/schema"
	"github.com/segmentio/kafka-go"
)

// ResumeWorker 封装 Kafka 消费者，并持有 ServiceContext
type ResumeWorker struct {
	svcCtx *svc.ServiceContext
}

// NewResumeWorker 构造函数
func NewResumeWorker(svcCtx *svc.ServiceContext) *ResumeWorker {
	return &ResumeWorker{
		svcCtx: svcCtx,
	}
}

// Start 启动消费者循环 (此方法会阻塞，所以外部需要用 goroutine 调用)
func (w *ResumeWorker) Start() {
	// 从 svcCtx 的 Config 中读取 Kafka 配置
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     w.svcCtx.Config.Kafka.Brokers,
		GroupID:     "resume-parser-group", // 或者从配置读取
		Topic:       w.svcCtx.Config.Kafka.Topic,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	ctx := context.Background() // 后台任务的根 Context
	fmt.Println("Kafka Resume Worker is running in background...")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("Error fetching message: %v", err)
			continue
		}

		var task mq.ParseTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			log.Printf("解析 JSON 失败的脏数据: %v", err)
			reader.CommitMessages(ctx, msg) // 脏数据直接提交丢弃
			continue
		}

		fmt.Printf("收到解析任务... ResumeID: %s\n", task.ResumeID)

		// 调用处理函数，把 svcCtx 和 ctx 传进去
		err = w.handleResumeParse(ctx, task)
		if err != nil {
			log.Printf("任务处理失败 (ResumeID: %s): %v", task.ResumeID, err)
			continue // 失败不提交，等下次重试
		}

		// 成功处理，提交 offset
		err = reader.CommitMessages(ctx, msg)
		if err != nil {
			log.Printf("Commit offset 失败: %v", err)
		} else {
			fmt.Printf("任务完成并提交 (Partition: %d, Offset: %d)\n", msg.Partition, msg.Offset)
		}
	}
}

// handleResumeParse 真正的业务处理逻辑
func (w *ResumeWorker) handleResumeParse(ctx context.Context, task mq.ParseTask) error {
	fmt.Printf("开始处理简历 - URL: %s, 类型: %s\n", task.FileURL, task.FileType)

	// 1. 下载文件
	fileBytes, err := w.svcCtx.Minio.DownloadFile(ctx, task.FileURL)
	if err != nil {
		return err
	}
	fmt.Println("-> 1. 文件下载完成")

	var parsedText string
	// 2. AI 提取文本
	if task.FileType == "png" || task.FileType == "jpg" || task.FileType == "jpeg" {
		parsedText, err = w.svcCtx.AI.ParseResumeToMarkdown(ctx, fileBytes, task.FileType)
		if err != nil {
			return err
		}
		fmt.Println("-> 2. 图片提为 Markdown 完成")
	} else if task.FileType == "pdf" {
		// 假设此处 PDF 提取出的仅是纯文本
		parsedText, err = pdfparser.ExtractTextFromPDF(fileBytes)
		if err != nil {
			return fmt.Errorf("PDF纯文本提取失败: %w", err)
		}
		fmt.Println("-> 2. PDF 文本提取完成")
	}

	chunks, err := splitDocument(ctx, parsedText, task.FileType, task.ResumeID)
	if err != nil {
		return fmt.Errorf("文档切片失败: %w", err)
	}
	fmt.Printf("-> 3. Eino 切片完成，共分成 %d 块\n", len(chunks))

	for _, chunk := range chunks {
		if chunk.MetaData == nil {
			chunk.MetaData = make(map[string]any)
		}
		// 注入你期望的额外元数据，Eino 会在入库时自动映射为 Qdrant 的 Payload

		chunk.MetaData["resume_id"] = task.ResumeID
	}

	// 见证魔法时刻：这一行代码搞定了调用大模型 Embedding 并存入 Qdrant
	ids, err := w.svcCtx.VectorStore.Indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("向量入库失败: %w", err)
	}

	fmt.Printf("-> 4. 向量入库完成，成功存入 %d 条数据，Qdrant 返回的 ID 列表: %v\n", len(ids), ids)
	// 5. 更新 MySQL 状态

	err = w.svcCtx.DB.UpdateResumeStatus(ctx, task.ResumeID, 2)
	if err != nil {
		return err
	}
	fmt.Println("-> 5. 状态更新完成")

	return nil
}

func splitDocument(ctx context.Context, parsedText string, fileType string, resumeID string) ([]*schema.Document, error) {
	// 1. 将原始文本包装成 Eino 需要的标准格式
	doc := &schema.Document{
		ID:      resumeID,
		Content: parsedText,
		MetaData: map[string]any{
			"file_type": fileType,
		},
	}
	srcDocs := []*schema.Document{doc}

	// 2. 根据类型选择切分策略
	if isMarkdown(fileType) {
		// --- 方案 A: Markdown 按标题切分 ---
		mdConfig := &markdown.HeaderConfig{
			Headers: map[string]string{
				"#":   "Level 1 Heading",
				"##":  "Level 2 Heading",
				"###": "Level 3 Heading",
			},
			TrimHeaders: false,
		}

		mdSplitter, err := markdown.NewHeaderSplitter(ctx, mdConfig)
		if err != nil {
			return nil, fmt.Errorf("初始化 Markdown 切分器失败: %w", err)
		}

		return mdSplitter.Transform(ctx, srcDocs)
	}

	// --- 方案 B: 纯文本按字符长度递归切分 ---
	recConfig := &recursive.Config{
		ChunkSize:   500, // 这里的块大小建议根据你所使用的 Embedding 模型的最大 Token 长度进行调整
		OverlapSize: 50,
		Separators:  []string{"\n\n", "\n", " ", ""},
		KeepType:    recursive.KeepTypeNone,
	}

	charSplitter, err := recursive.NewSplitter(ctx, recConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 Recursive 切分器失败: %w", err)
	}

	return charSplitter.Transform(ctx, srcDocs)
}

// isMarkdown 辅助判断当前文本结构是否倾向于使用 Markdown 切分器
func isMarkdown(fileType string) bool {
	// 这里假设你大模型提取图片输出的格式是包含 Markdown Header 语法的
	return fileType == "md"
}
