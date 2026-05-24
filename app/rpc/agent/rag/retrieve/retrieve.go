package retrieve

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/agent"
	"Goffer/pkg/convert"
	"Goffer/pkg/logger"
	"context"
	"fmt"

	qdrant_ret "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

type RetrieveService struct {
	svc        *svc.ServiceContext
	retrievers map[string]retriever.Retriever
}

func NewRetrieveService(svc *svc.ServiceContext) *RetrieveService {
	ctx := context.Background()

	repo := &RetrieveService{
		svc:        svc,
		retrievers: make(map[string]retriever.Retriever),
	}

	collections := []string{"goffer_resumes", "goffer_question_bank", "goffer_jd_bank"}
	for _, col := range collections {
		ret, err := qdrant_ret.NewRetriever(ctx, &qdrant_ret.Config{
			Client:     svc.QdrantClient,
			Collection: col,
			Embedding:  svc.Embedder,
			TopK:       3,
		})
		if err != nil {
			logger.Error("初始化集合的 Retriever 失败",
				zap.String("collection", col),
				zap.Error(err),
			)
		}

		repo.retrievers[col] = ret
	}

	return repo
}

func (s *RetrieveService) RetrieveContext(ctx context.Context, req *agent.RetrieveReq) (*agent.RetrieveResp, error) {
	logger.InfoCtx(ctx, "收到检索请求",
		zap.String("user_id", req.UserId),
		zap.String("query", req.Query),
		zap.String("collection", req.Collection),
	)

	ret, exists := s.retrievers[req.Collection]
	if !exists {
		return nil, fmt.Errorf("未注册或不支持的集合: %s", req.Collection)
	}

	var opts []retriever.Option
	var mustConditions []*qdrant.Condition

	if req.TopK != nil && *req.TopK > 0 {
		opts = append(opts, retriever.WithTopK(int(*req.TopK)))
	}

	switch req.Collection {
	case "goffer_resumes":
		if req.ResumeId == nil || *req.ResumeId == "" {
			return nil, fmt.Errorf("非法检索：检索简历切片必须提供确切的 ResumeID")
		}
		mustConditions = append(mustConditions, buildKeywordCondition("resume_id", *req.ResumeId))

	case "goffer_question_bank":
		if req.Difficulty != nil && *req.Difficulty != "" {
			mustConditions = append(mustConditions, buildKeywordCondition("difficulty", *req.Difficulty))
		}

		if len(req.Tags) > 0 {
			tagsOrCondition := buildAnyKeywordsCondition("tags", req.Tags)
			mustConditions = append(mustConditions, tagsOrCondition)
			logger.InfoCtx(ctx, "智能题库多标签检索",
				zap.String("difficulty", *req.Difficulty),
				zap.Strings("tags", req.Tags),
			)
		}
		logger.InfoCtx(ctx, "智能题库高级过滤")

	case "goffer_jd_bank":
		if req.Company != nil && *req.Company != "" {
			mustConditions = append(mustConditions, buildKeywordCondition("company", *req.Company))
		}
		logger.InfoCtx(ctx, "岗位对齐检索 JD 库")

	default:
		logger.WarnCtx(ctx, "检索了未注册的 Collection", zap.String("collection", req.Collection))
	}

	if len(mustConditions) > 0 {
		filter := &qdrant.Filter{
			Must: mustConditions,
		}
		opts = append(opts, qdrant_ret.WithFilter(filter))
	}

	docs, err := ret.Retrieve(ctx, req.Query, opts...)
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	var contexts []string
	var metaInfo string
	for _, doc := range docs {
		if req.Collection == "goffer_question_bank" {
			tags := convert.GetStringFromMeta(doc, "tags")
			difficulty := convert.GetStringFromMeta(doc, "difficulty")
			metaInfo = fmt.Sprintf("[标签:%s | 难度:%s]\n", tags, difficulty)
		} else if req.Collection == "goffer_jd_bank" {
			company := convert.GetStringFromMeta(doc, "company")
			title := convert.GetStringFromMeta(doc, "title")
			metaInfo = fmt.Sprintf("[公司:%s | 岗位:%s]\n", company, title)
		}

		contexts = append(contexts, metaInfo+doc.Content)
	}

	logger.InfoCtx(ctx, "检索完成", zap.Int("hit_count", len(contexts)))

	return &agent.RetrieveResp{
		Contexts: contexts,
	}, nil
}

func buildKeywordCondition(key string, value string) *qdrant.Condition {
	return &qdrant.Condition{
		ConditionOneOf: &qdrant.Condition_Field{
			Field: &qdrant.FieldCondition{
				Key: key,
				Match: &qdrant.Match{
					MatchValue: &qdrant.Match_Keyword{
						Keyword: value,
					},
				},
			},
		},
	}
}

func buildAnyKeywordsCondition(key string, values []string) *qdrant.Condition {
	return &qdrant.Condition{
		ConditionOneOf: &qdrant.Condition_Field{
			Field: &qdrant.FieldCondition{
				Key: key,
				Match: &qdrant.Match{
					MatchValue: &qdrant.Match_Keywords{
						Keywords: &qdrant.RepeatedStrings{
							Strings: values,
						},
					},
				},
			},
		},
	}
}
