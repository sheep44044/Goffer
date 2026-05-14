package retrieve

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/agent"
	"context"
	"fmt"

	qdrant_ret "github.com/cloudwego/eino-ext/components/retriever/qdrant"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/qdrant/go-client/qdrant"
)

type RetrieveService struct {
	svc *svc.ServiceContext
}

func NewRetrieveService(svc *svc.ServiceContext) *RetrieveService {
	return &RetrieveService{
		svc: svc,
	}
}

// RetrieveContext 是提供给 Interview 调用的 RPC 方法
func (s *RetrieveService) RetrieveContext(ctx context.Context, req *agent.RetrieveReq) (*agent.RetrieveResp, error) {
	fmt.Printf("[RAG 服务] 收到检索请求 - UserID: %s, Query: %s\n", req.UserId, req.Query)

	topK := 3
	if req.TopK != nil && *req.TopK > 0 {
		topK = int(*req.TopK)
	}

	var opts []retriever.Option
	var mustConditions []*qdrant.Condition

	// 1. 动态构建 Qdrant 的 Metadata 过滤条件
	switch req.Collection {
	case "goffer_resumes":
		// 修正：改为通过 resume_id 过滤。前提是调用方（面试模块）要把正在面试的 resume_id 传过来
		if req.ResumeId != nil && *req.ResumeId != "" {
			mustConditions = append(mustConditions, buildKeywordCondition("resume_id", *req.ResumeId))
			fmt.Printf("-> [精准命中] 锁定特定简历: %s\n", *req.ResumeId)
		} else {
			fmt.Println("-> [警告] 检索简历时未提供 ResumeID，可能搜到其他人的切片！")
		}

	case "goffer_question_bank":
		// 🚀 进阶：如果只想抽取 "中等" 难度的题目
		if req.Difficulty != nil && *req.Difficulty != "" {
			mustConditions = append(mustConditions, buildKeywordCondition("difficulty", *req.Difficulty))
		}
		// 🚀 进阶：如果只想检索 "Golang" 标签的题
		// (假设只传一个主标签，实际多标签匹配可用 qdrant.Match_Any)
		if req.Tag != nil && *req.Tag != "" {
			mustConditions = append(mustConditions, buildKeywordCondition("tags", *req.Tag))
		}
		fmt.Println("-> [智能题库] 使用题库高级过滤")

	case "goffer_jd_bank": // 🚨 修正：对其 jd_store.go 里的集合名称
		// 🚀 进阶：如果面试官要求根据特定公司（比如字节跳动）的 JD 风格来提问
		if req.Company != nil && *req.Company != "" {
			mustConditions = append(mustConditions, buildKeywordCondition("company", *req.Company))
		}
		fmt.Println("-> [岗位对齐] 检索 JD 库")

	default:
		fmt.Printf("-> [警告] 检索了未注册的 Collection: %s\n", req.Collection)
	}

	// 2. 将拼装好的条件应用到 Retriever
	if len(mustConditions) > 0 {
		filter := &qdrant.Filter{
			Must: mustConditions,
		}
		opts = append(opts, qdrant_ret.WithFilter(filter))
	}

	// 3. 初始化 Eino 的 Qdrant Retriever
	ret, err := qdrant_ret.NewRetriever(ctx, &qdrant_ret.Config{
		Client:     s.svc.QdrantClient,
		Collection: req.Collection,
		TopK:       topK,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化检索器失败: %w", err)
	}

	// 4. 执行向量检索
	docs, err := ret.Retrieve(ctx, req.Query, opts...)
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	// 5. 将 Eino 的 Document 转换为纯文本返回
	var contexts []string
	for _, doc := range docs {
		// 增加可读性：把元数据也拼进去还给大模型
		metaInfo := ""
		if req.Collection == "goffer_question_bank" {
			metaInfo = fmt.Sprintf("[标签:%v | 难度:%v]\n", doc.MetaData["tags"], doc.MetaData["difficulty"])
		} else if req.Collection == "goffer_jd_bank" {
			metaInfo = fmt.Sprintf("[公司:%v | 岗位:%v]\n", doc.MetaData["company"], doc.MetaData["title"])
		}

		contexts = append(contexts, metaInfo+doc.Content)
	}

	fmt.Printf("-> 检索完成，共命中 %d 条上下文\n", len(contexts))

	return &agent.RetrieveResp{
		Contexts: contexts,
	}, nil
}

// 辅助函数：快速构建 Qdrant 的精确匹配过滤条件
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
