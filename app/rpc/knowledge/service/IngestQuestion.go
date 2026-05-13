package service

import (
	"Goffer/app/rpc/knowledge/dal/db"
	"Goffer/app/rpc/knowledge/mq"
	"Goffer/app/rpc/knowledge/svc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"Goffer/pkg/snowflake"
	"context"
	"fmt"
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
	fmt.Println("[Knowledge 服务] 开始处理题库入库...")

	difficulty := "未定级"
	if req.Difficulty != nil && *req.Difficulty != "" {
		difficulty = *req.Difficulty
	}

	var tags []string
	if req.Tags != nil && len(req.Tags) > 0 {
		tags = req.Tags
	}

	if len(tags) == 0 || difficulty == "未定级" {
		fmt.Println("-> 触发 AI 自动抽取标签与难度...")
		// 调用你封装好的大模型接口（这里建议用响应极快的小模型，如 doubao-lite / gpt-4o-mini）
		aiResult, err := s.svc.AI.GenerateTagsAndDifficulty(ctx, req.QuestionContent, req.StandardAnswer)
		if err != nil {
			// 如果 AI 调用失败，为了不阻塞业务，可以记录日志并使用默认值，而不是直接 return err
			fmt.Printf("[警告] AI 抽取失败，使用默认值: %v\n", err)
		} else {
			// 用 AI 的结果覆盖空值
			if len(tags) == 0 {
				tags = aiResult.Tags
			}
			if difficulty == "未定级" {
				difficulty = aiResult.Difficulty
			}
		}
	}

	questionID := snowflake.GenString()
	err := s.svc.DB.CreateQuestion(ctx, []*db.Question{{
		QuestionID:      questionID,
		QuestionContent: req.QuestionContent,
		StandardAnswer:  req.StandardAnswer,
		Difficulty:      difficulty,
		Tags:            tags,
	}})
	if err != nil {
		return "", fmt.Errorf("写入 MySQL 失败: %w", err)
	}
	fmt.Printf("-> MySQL 写入成功，生成题库ID: %s\n", questionID)

	err = s.svc.Kafka.SendQuestionParseTask(mq.QuestionParseTask{
		QuestionID:      questionID,
		QuestionContent: req.QuestionContent,
		StandardAnswer:  req.StandardAnswer,
		Difficulty:      difficulty,
		Tags:            tags,
	})
	if err != nil {
		return "", fmt.Errorf("kafka publish failed: %w", errno.ServiceErr.WithMessage("解析任务投递失败，请稍后重试"))
	}

	return questionID, nil
}
