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

type QuestionService struct {
	svc *svc.ServiceContext
}

func NewQuestionService(svc *svc.ServiceContext) *QuestionService {
	return &QuestionService{
		svc: svc,
	}
}

func (s *QuestionService) IngestQuestion(ctx context.Context, req *knowledge.IngestQuestionReq) (string, error) {
	logger.InfoCtx(ctx, "开始处理题库入库")

	difficulty := "未定级"
	if req.Difficulty != nil && *req.Difficulty != "" {
		difficulty = *req.Difficulty
	}

	var tags []string
	if req.Tags != nil && len(req.Tags) > 0 {
		tags = req.Tags
	}

	if len(tags) == 0 || difficulty == "未定级" {
		logger.InfoCtx(ctx, "触发 AI 自动抽取标签与难度")
		aiResult, err := s.svc.AI.GenerateTagsAndDifficulty(ctx, req.QuestionContent, req.StandardAnswer)
		if err != nil {
			logger.WarnCtx(ctx, "AI 抽取失败，使用默认值", zap.Error(err))
		} else {
			if len(tags) == 0 {
				tags = aiResult.Tags
			}
			if difficulty == "未定级" {
				difficulty = aiResult.Difficulty
			}
		}
	}

	questionID := snowflake.GenString()
	if err := s.svc.DB.CreateQuestion(ctx, []*db.Question{{
		QuestionID:      questionID,
		QuestionContent: req.QuestionContent,
		StandardAnswer:  req.StandardAnswer,
		Difficulty:      difficulty,
		Tags:            tags,
	}}); err != nil {
		return "", fmt.Errorf("写入 MySQL 失败: %w", err)
	}
	logger.InfoCtx(ctx, "题库 MySQL 写入成功", zap.String("question_id", questionID))

	if err := s.svc.Kafka.SendQuestionParseTask(ctx, mq.QuestionParseTask{
		QuestionID:      questionID,
		QuestionContent: req.QuestionContent,
		StandardAnswer:  req.StandardAnswer,
		Difficulty:      difficulty,
		Tags:            tags,
	}); err != nil {
		logger.ErrorCtx(ctx, "Kafka 投递题库解析任务失败", zap.String("question_id", questionID), zap.Error(err))
		return "", errno.ServiceErr.WithMessage("解析任务投递失败，请稍后重试")
	}

	return questionID, nil
}
