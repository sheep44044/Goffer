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

type JDCSVService struct {
	svc *svc.ServiceContext
}

func NewJDCSVService(svc *svc.ServiceContext) *JDCSVService {
	return &JDCSVService{
		svc: svc,
	}
}

// UploadJDCSV 处理 JD 的 CSV 批量导入
func (s *JDCSVService) UploadJDCSV(ctx context.Context, req *knowledge.UploadJDReq) (string, string, error) {
	// 1. 上传至 MinIO
	safeFileName, fileURL, err := s.svc.Minio.UploadFile(ctx, req.FileName, req.FileContent, req.ContentType)
	if err != nil {
		return "", "", fmt.Errorf("上传 JD CSV 到 MinIO 失败: %w", err)
	}
	fmt.Printf("JD CSV 已备份至 MinIO: %s\n", fileURL)

	// 2. 记录导入批次
	csvID := snowflake.GenString()
	err = s.svc.DB.CreateJDCSV(ctx, []*db.JDCSV{{
		ID:       csvID,
		UserID:   req.UserId,
		FileURL:  fileURL,
		FileName: safeFileName,
	}})
	if err != nil {
		return "", "", fmt.Errorf("创建 JD CSV 导入记录失败: %w", err)
	}

	// 3. 开始解析 CSV 内容
	reader := csv.NewReader(bytes.NewReader(req.FileContent))
	_, err = reader.Read() // 跳过表头
	if err != nil {
		return "", "", fmt.Errorf("读取 CSV 表头失败: %w", err)
	}

	var dbJDs []*db.JD

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("解析 CSV 行失败: %v\n", err)
			continue
		}

		// JD 数据至少需要有 公司(0)、岗位(1)、职责(2)、要求(3) 才能算作一条有效的 JD
		// 这里加入简单的非空校验，如果公司或岗位名为空，跳过此行
		if len(record) < 4 || strings.TrimSpace(record[0]) == "" || strings.TrimSpace(record[1]) == "" {
			continue
		}

		company := record[0]
		title := record[1]
		responsibilities := record[2]
		requirements := record[3]

		// 处理第 5 列的 Tags（如果有）
		var tags []string
		if len(record) > 4 && strings.TrimSpace(record[4]) != "" {
			rawTagsStr := strings.ReplaceAll(record[4], "，", ",")
			rawTags := strings.Split(rawTagsStr, ",")

			for _, t := range rawTags {
				cleanTag := strings.TrimSpace(t)
				if cleanTag != "" {
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

		err = s.svc.Kafka.SendJDParseTask(mq.JDParseTask{
			JDID:             jdID,
			Company:          company,
			Title:            title,
			Responsibilities: responsibilities,
			Requirements:     requirements,
			Tags:             tags,
		})

		if err != nil {
			fmt.Printf("投递 JD 到 Kafka 失败(ID:%s): %v\n", jdID, err)
		}
	}

	if len(dbJDs) > 0 {
		err = s.svc.DB.CreateJD(ctx, dbJDs)
		if err != nil {
			return "", "", fmt.Errorf("批量写入 MySQL 失败: %w", err)
		}
	}

	return csvID, fileURL, nil
}
