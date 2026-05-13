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

type JDService struct {
	svc *svc.ServiceContext
}

func NewJDService(svc *svc.ServiceContext) *JDService {
	return &JDService{
		svc: svc,
	}
}

func (s *JDService) IngestQuestion(ctx context.Context, req *knowledge.IngestJDReq) (string, error) {
	fmt.Println("[Knowledge 服务] 开始处理 JD (职位描述) 入库...")

	var tags []string
	if req.Tags != nil && len(req.Tags) > 0 {
		tags = req.Tags
	}

	if len(tags) == 0 {
		fmt.Println("-> 触发 AI 自动抽取 JD 标签...")

		// 将 JD 的核心内容拼接起来，喂给大模型提取标签（例如：Java, 微服务, 高并发）
		jdContent := fmt.Sprintf("职位:%s\n职责:%s\n要求:%s", req.Title, req.Responsibilities, req.Requirements)

		// 注意：这里你需要实现一个类似 GenerateJDTags 的方法，或者复用之前的方法但修改 Prompt
		aiTags, err := s.svc.AI.GenerateJDTags(ctx, jdContent)
		if err != nil {
			fmt.Printf("[警告] AI 抽取 JD 标签失败，使用空标签: %v\n", err)
		} else {
			tags = aiTags
		}
	}

	jdID := snowflake.GenString()

	err := s.svc.DB.CreateJD(ctx, []*db.JD{{
		JDID:             jdID,
		Company:          req.Company,
		Title:            req.Title,
		Responsibilities: req.Responsibilities,
		Requirements:     req.Requirements,
		Tags:             tags,
	}})
	if err != nil {
		return "", fmt.Errorf("写入 MySQL 失败: %w", err)
	}
	fmt.Printf("-> MySQL 写入成功，生成 JD_ID: %s\n", jdID)

	// 3. 投递到 Kafka 给 RAG 服务做向量化
	err = s.svc.Kafka.SendJDParseTask(mq.JDParseTask{
		JDID:             jdID,
		Company:          req.Company,
		Title:            req.Title,
		Responsibilities: req.Responsibilities,
		Requirements:     req.Requirements,
		Tags:             tags,
	})
	if err != nil {
		return "", fmt.Errorf("kafka publish failed: %w", errno.ServiceErr.WithMessage("JD 解析任务投递失败，请稍后重试"))
	}

	return jdID, nil
}
