package resume

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/logger"
	"Goffer/pkg/pdfparser"
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	qdrant_indexer "github.com/cloudwego/eino-ext/components/indexer/qdrant"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

type ResumeWorker struct {
	svc     *svc.ServiceContext
	indexer indexer.Indexer
}

func NewResumeWorker(svc *svc.ServiceContext) (*ResumeWorker, error) {
	ctx := context.Background()

	idx, err := qdrant_indexer.NewIndexer(ctx, &qdrant_indexer.Config{
		Client:     svc.QdrantClient,
		Collection: "goffer_resumes",
		Embedding:  svc.Embedder,
		VectorDim:  2048,
		Distance:   qdrant.Distance_Cosine,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 Resume Indexer 失败: %w", err)
	}

	return &ResumeWorker{
		svc:     svc,
		indexer: idx,
	}, nil
}

type ResumeParseTask struct {
	ResumeID string `json:"resume_id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
}

func (w *ResumeWorker) HandleResumeParse(ctx context.Context, task ResumeParseTask) error {
	logger.InfoCtx(ctx, "开始处理简历",
		zap.String("file_url", task.FileURL),
		zap.String("file_type", task.FileType),
	)

	fileBytes, err := w.svc.Minio.DownloadFile(ctx, task.FileURL)
	if err != nil {
		return err
	}
	logger.InfoCtx(ctx, "文件下载完成")

	var parsedText string
	if task.FileType == "image/png" || task.FileType == "image/jpg" || task.FileType == "image/jpeg" {
		parsedText, err = w.svc.AI.ParseResumeToMarkdown(ctx, fileBytes, task.FileType)
		if err != nil {
			return err
		}
		logger.InfoCtx(ctx, "图片转为 Markdown 完成")
	} else if task.FileType == "pdf" {
		parsedText, err = pdfparser.ExtractTextFromPDF(fileBytes)
		if err != nil {
			return fmt.Errorf("PDF纯文本提取失败: %w", err)
		}
		logger.InfoCtx(ctx, "PDF 文本提取完成")
	}

	chunks, err := splitDocument(ctx, parsedText, task.FileType, task.ResumeID)
	if err != nil {
		return fmt.Errorf("文档切片失败: %w", err)
	}
	logger.InfoCtx(ctx, "Eino 切片完成", zap.Int("chunk_count", len(chunks)))

	for _, chunk := range chunks {
		if chunk.MetaData == nil {
			chunk.MetaData = make(map[string]any)
		}
		chunk.MetaData["resume_id"] = task.ResumeID
	}

	ids, err := w.indexer.Store(ctx, chunks)
	if err != nil {
		return fmt.Errorf("向量入库失败: %w", err)
	}
	logger.InfoCtx(ctx, "向量入库完成", zap.Int("stored_count", len(ids)))

	_, err = w.svc.UserClient.UpdateResumeStatus(ctx, &user.UpdateResumeStatusReq{
		ResumeId: task.ResumeID,
		Status:   2,
	})
	if err != nil {
		logger.WarnCtx(ctx, "Qdrant入库成功但回调User状态失败，需手动修复",
			zap.String("resume_id", task.ResumeID),
			zap.Error(err),
		)
	}
	return nil
}

func splitDocument(ctx context.Context, parsedText string, fileType string, resumeID string) ([]*schema.Document, error) {
	doc := &schema.Document{
		ID:      resumeID,
		Content: parsedText,
		MetaData: map[string]any{
			"file_type": fileType,
		},
	}
	srcDocs := []*schema.Document{doc}

	if isMarkdown(fileType) {
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

	recConfig := &recursive.Config{
		ChunkSize:   500,
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

func isMarkdown(fileType string) bool {
	return fileType == "md"
}
