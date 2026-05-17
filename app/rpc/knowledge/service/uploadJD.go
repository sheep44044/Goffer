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

type JDCSVService struct {
	svc *svc.ServiceContext
}

func NewJDCSVService(svc *svc.ServiceContext) *JDCSVService {
	return &JDCSVService{svc: svc}
}

func (s *JDCSVService) UploadJDCSV(ctx context.Context, req *knowledge.UploadJDReq) (string, string, error) {
	safeFileName, fileURL, err := s.svc.Minio.UploadFile(ctx, req.FileName, req.FileContent, req.ContentType)
	if err != nil {
		return "", "", fmt.Errorf("上传 JD CSV 到 MinIO 失败: %w", err)
	}
	logger.InfoCtx(ctx, "JD CSV 已备份至 MinIO", zap.String("file_url", fileURL))

	csvID := snowflake.GenString()
	if err = s.svc.DB.CreateJDCSV(ctx, []*db.JDCSV{{
		ID:       csvID,
		UserID:   req.UserId,
		FileURL:  fileURL,
		FileName: safeFileName,
	}}); err != nil {
		return "", "", fmt.Errorf("创建 JD CSV 导入记录失败: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(req.FileContent))
	if _, err = reader.Read(); err != nil {
		return "", "", fmt.Errorf("读取 CSV 表头失败: %w", err)
	}

	var dbJDs []*db.JD
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.WarnCtx(ctx, "解析 CSV 行失败", zap.Error(err))
			continue
		}

		if len(record) < 4 || strings.TrimSpace(record[0]) == "" || strings.TrimSpace(record[1]) == "" {
			continue
		}

		company := record[0]
		title := record[1]
		responsibilities := record[2]
		requirements := record[3]

		var tags []string
		if len(record) > 4 && strings.TrimSpace(record[4]) != "" {
			rawTagsStr := strings.ReplaceAll(record[4], ",", ",")
			for _, t := range strings.Split(rawTagsStr, ",") {
				if cleanTag := strings.TrimSpace(t); cleanTag != "" {
					tags = append(tags, cleanTag)
				}
			}
		}

		jdID := snowflake.GenString()
		dbJDs = append(dbJDs, &db.JD{
			JDID:             jdID,
			Company:          company,
			Title:            title,
			Responsibilities: responsibilities,
			Requirements:     requirements,
			Tags:             tags,
			CSVID:            csvID,
		})

		if err := s.svc.Kafka.SendJDParseTask(ctx, mq.JDParseTask{
			JDID:             jdID,
			Company:          company,
			Title:            title,
			Responsibilities: responsibilities,
			Requirements:     requirements,
			Tags:             tags,
		}); err != nil {
			logger.ErrorCtx(ctx, "投递 JD 到 Kafka 失败", zap.String("jd_id", jdID), zap.Error(err))
		}
	}

	if len(dbJDs) > 0 {
		if err = s.svc.DB.CreateJD(ctx, dbJDs); err != nil {
			return "", "", fmt.Errorf("批量写入 MySQL 失败: %w", err)
		}
	}

	return csvID, fileURL, nil
}
