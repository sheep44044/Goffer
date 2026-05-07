package worker

import (
	"Goffer/app/rpc/user/mq"
	"Goffer/pkg/pdfparser"
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/schema"
)

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
