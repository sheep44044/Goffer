package service

import (
	"Goffer/app/rpc/knowledge/dal/db"
	"Goffer/app/rpc/knowledge/mq"
	"Goffer/app/rpc/knowledge/svc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"Goffer/pkg/snowflake"
	"context"
	"fmt"

	"go.uber.org/zap"
)

type JDService struct {
	svc *svc.ServiceContext
}

func NewJDService(svc *svc.ServiceContext) *JDService {
	return &JDService{
		svc: svc,
	}
}

func (s *JDService) IngestQuestion(ctx context.Context, req *knowledge.IngestJDReq) (string, error) {
	logger.InfoCtx(ctx, "开始处理 JD 入库")

	var tags []string
	if req.Tags != nil && len(req.Tags) > 0 {
		tags = req.Tags
	}

	if len(tags) == 0 {
		logger.InfoCtx(ctx, "触发 AI 自动抽取 JD 标签")
		jdContent := fmt.Sprintf("职位:%s\n职责:%s\n要求:%s", req.Title, req.Responsibilities, req.Requirements)
		aiTags, err := s.svc.AI.GenerateJDTags(ctx, jdContent)
		if err != nil {
			logger.WarnCtx(ctx, "AI 抽取 JD 标签失败，使用空标签", zap.Error(err))
		} else {
			tags = aiTags
		}
	}

	jdID := snowflake.GenString()
	if err := s.svc.DB.CreateJD(ctx, []*db.JD{{
		JDID:             jdID,
		Company:          req.Company,
		Title:            req.Title,
		Responsibilities: req.Responsibilities,
		Requirements:     req.Requirements,
		Tags:             tags,
	}}); err != nil {
		return "", fmt.Errorf("写入 MySQL 失败: %w", err)
	}
	logger.InfoCtx(ctx, "JD MySQL 写入成功", zap.String("jd_id", jdID))

	if err := s.svc.Kafka.SendJDParseTask(ctx, mq.JDParseTask{
		JDID:             jdID,
		Company:          req.Company,
		Title:            req.Title,
		Responsibilities: req.Responsibilities,
		Requirements:     req.Requirements,
		Tags:             tags,
	}); err != nil {
		logger.ErrorCtx(ctx, "Kafka 投递 JD 解析任务失败", zap.String("jd_id", jdID), zap.Error(err))
		return "", errno.ServiceErr.WithMessage("JD 解析任务投递失败，请稍后重试")
	}

	return jdID, nil
}
