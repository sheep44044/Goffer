package service

import (
	"Goffer/app/rpc/knowledge/dal/db"
	"Goffer/app/rpc/knowledge/mq"
	"Goffer/app/rpc/knowledge/svc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/snowflake"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type QuestionCSVService struct {
	svc *svc.ServiceContext
}

func NewQuestionCSVService(svc *svc.ServiceContext) *QuestionCSVService {
	return &QuestionCSVService{
		svc: svc,
	}
}

func (s *QuestionCSVService) UploadQuestionCSV(ctx context.Context, req *knowledge.UploadQuestionReq) (string, string, error) {
	safeFileName, fileURL, err := s.svc.Minio.UploadFile(ctx, req.FileName, req.FileContent, req.ContentType)
	if err != nil {
		return "", "", fmt.Errorf("上传 CSV 到 MinIO 失败: %w", err)
	}
	fmt.Printf("CSV 已备份至 MinIO: %s\n", fileURL)

	csvID := snowflake.GenString()
	err = s.svc.DB.CreateQuestionCSV(ctx, []*db.QuestionCSV{{
		ID:       csvID,
		UserID:   req.UserId,
		FileURL:  fileURL,
		FileName: safeFileName,
	}})
	if err != nil {
		return "", "", fmt.Errorf("db create resume failed: %w", err)
	}

	// 3. 开始解析 CSV 内容
	reader := csv.NewReader(bytes.NewReader(req.FileContent))
	_, err = reader.Read() // 跳过表头
	if err != nil {
		return "", "", fmt.Errorf("读取 CSV 表头失败: %w", err)
	}

	var dbQuestions []*db.Question

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("解析 CSV 行失败: %v\n", err)
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
			rawTagsStr := strings.ReplaceAll(record[3], "，", ",")
			rawTags := strings.Split(rawTagsStr, ",")

			for _, t := range rawTags {
				cleanTag := strings.TrimSpace(t)
				if cleanTag != "" {
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

		err = s.svc.Kafka.SendQuestionParseTask(mq.QuestionParseTask{
			QuestionID:      questionID,
			QuestionContent: questionContent,
			StandardAnswer:  answer,
			Difficulty:      difficulty,
			Tags:            tags,
		})

		if err != nil {
			fmt.Printf("投递题目到 Kafka 失败(ID:%s): %v\n", questionID, err)
		}
	}

	if len(dbQuestions) > 0 {
		err = s.svc.DB.CreateQuestion(ctx, dbQuestions)
		if err != nil {
			return "", "", fmt.Errorf("批量写入 MySQL 失败: %w", err)
		}
	}

	return csvID, fileURL, nil
}
