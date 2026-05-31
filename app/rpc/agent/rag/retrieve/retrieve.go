package retrieve

import (
	"Goffer/app/rpc/agent/rag/rerank"
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/agent"
	"Goffer/pkg/convert"
	"Goffer/pkg/logger"
	"context"
	"fmt"

	qdrant_ret "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

type RetrieveService struct {
	svc        *svc.ServiceContext
	retrievers map[string]retriever.Retriever
	reranker   document.Transformer // ScoreReranker: 召回后按首因/近因效应重排
}

func NewRetrieveService(svc *svc.ServiceContext) *RetrieveService {
	ctx := context.Background()

	// 初始化 ScoreReranker（基于 Qdrant 返回的余弦相似度分数）
	rr, err := rerank.NewReranker(ctx, nil)
	if err != nil {
		logger.Error("初始化 Reranker 失败，检索将跳过重排", zap.Error(err))
	}

	repo := &RetrieveService{
		svc:        svc,
		retrievers: make(map[string]retriever.Retriever),
		reranker:   rr,
	}

	// 默认 TopK=10 给 Reranker 留足候选池，最终截断在 RetrieveContext 中完成
	collections := []string{"goffer_resumes", "goffer_question_bank", "goffer_jd_bank"}
	for _, col := range collections {
		ret, err := qdrant_ret.NewRetriever(ctx, &qdrant_ret.Config{
			Client:     svc.QdrantClient,
			Collection: col,
			Embedding:  svc.Embedder,
			TopK:       10,
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

	// 确定最终返回数和召回候选数（Rerank 需要在更大候选池上工作）
	finalTopK := 3
	if req.TopK != nil && *req.TopK > 0 {
		finalTopK = int(*req.TopK)
	}
	recallTopK := finalTopK * 3
	if recallTopK < 10 {
		recallTopK = 10
	}

	var opts []retriever.Option
	var mustConditions []*qdrant.Condition

	// 召回阶段使用扩大的 TopK
	opts = append(opts, retriever.WithTopK(recallTopK))

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

	// 粗筛召回
	docs, err := ret.Retrieve(ctx, req.Query, opts...)
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	logger.InfoCtx(ctx, "粗筛召回完成",
		zap.Int("recall_count", len(docs)),
		zap.Int("final_topk", finalTopK),
	)

	// Rerank 重排
	if s.reranker != nil && len(docs) > 1 {
		docs, err = s.reranker.Transform(ctx, docs)
		if err != nil {
			logger.WarnCtx(ctx, "Rerank 失败，回退到原始排序", zap.Error(err))
		} else {
			logger.InfoCtx(ctx, "Rerank 完成", zap.Int("doc_count", len(docs)))
		}
	}

	// 截取最终 TopK
	if len(docs) > finalTopK {
		docs = docs[:finalTopK]
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
