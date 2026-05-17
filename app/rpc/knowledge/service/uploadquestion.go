package service

import (
	"Goffer/app/rpc/knowledge/dal/db"
	"Goffer/app/rpc/knowledge/mq"
	"Goffer/app/rpc/knowledge/svc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/logger"
	"Goffer/pkg/snowflake"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
)

type QuestionCSVService struct {
	svc *svc.ServiceContext
}

func NewQuestionCSVService(svc *svc.ServiceContext) *QuestionCSVService {
	return &QuestionCSVService{svc: svc}
}

func (s *QuestionCSVService) UploadQuestionCSV(ctx context.Context, req *knowledge.UploadQuestionReq) (string, string, error) {
	safeFileName, fileURL, err := s.svc.Minio.UploadFile(ctx, req.FileName, req.FileContent, req.ContentType)
	if err != nil {
		return "", "", fmt.Errorf("上传 CSV 到 MinIO 失败: %w", err)
	}
	logger.InfoCtx(ctx, "题库 CSV 已备份至 MinIO", zap.String("file_url", fileURL))

	csvID := snowflake.GenString()
	if err = s.svc.DB.CreateQuestionCSV(ctx, []*db.QuestionCSV{{
		ID:       csvID,
		UserID:   req.UserId,
		FileURL:  fileURL,
		FileName: safeFileName,
	}}); err != nil {
		return "", "", fmt.Errorf("创建题库 CSV 导入记录失败: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(req.FileContent))
	if _, err = reader.Read(); err != nil {
		return "", "", fmt.Errorf("读取 CSV 表头失败: %w", err)
	}

	var dbQuestions []*db.Question
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.WarnCtx(ctx, "解析 CSV 行失败", zap.Error(err))
			continue
		}

		if len(record) < 2 || record[0] == "" {
			continue
		}

		questionContent := record[0]
		answer := record[1]
		difficulty := "未定级"
		if len(record) > 2 && record[2] != "" {
			difficulty = record[2]
		}

		var tags []string
		if len(record) > 3 && strings.TrimSpace(record[3]) != "" {
			rawTagsStr := strings.ReplaceAll(record[3], ",", ",")
			for _, t := range strings.Split(rawTagsStr, ",") {
				if cleanTag := strings.TrimSpace(t); cleanTag != "" {
					tags = append(tags, cleanTag)
				}
			}
		}

		questionID := snowflake.GenString()
		dbQuestions = append(dbQuestions, &db.Question{
			QuestionID:      questionID,
			QuestionContent: questionContent,
			StandardAnswer:  answer,
			Difficulty:      difficulty,
			Tags:            tags,
			CSVID:           csvID,
		})

		if err := s.svc.Kafka.SendQuestionParseTask(ctx, mq.QuestionParseTask{
			QuestionID:      questionID,
			QuestionContent: questionContent,
			StandardAnswer:  answer,
			Difficulty:      difficulty,
			Tags:            tags,
		}); err != nil {
			logger.ErrorCtx(ctx, "投递题目到 Kafka 失败", zap.String("question_id", questionID), zap.Error(err))
		}
	}

	if len(dbQuestions) > 0 {
		if err = s.svc.DB.CreateQuestion(ctx, dbQuestions); err != nil {
			return "", "", fmt.Errorf("批量写入 MySQL 失败: %w", err)
		}
	}

	return csvID, fileURL, nil
}
