package tools

import (
	"Goffer/app/rpc/agent/rag/retrieve"
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/agent"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type ResumeSkillInput struct {
	Query    string `json:"query" jsonschema:"description=要查询的简历关键词，如'微服务','Kafka'"`
	ResumeID string `json:"resume_id" jsonschema:"description=候选人的简历ID,required=true"`
}

func NewRetrieveResumeTool(svc *svc.ServiceContext) (tool.BaseTool, error) {
	return utils.InferTool("RetrieveResumeSkill", "根据用户的回答，检索该候选人简历中的项目细节进行验证",
		func(ctx context.Context, input *ResumeSkillInput) (string, error) {
			topK := int32(3)
			resp, err := retrieve.NewRetrieveService(svc).RetrieveContext(ctx, &agent.RetrieveReq{
				Query:      input.Query,
				Collection: "goffer_resumes",
				ResumeId:   &input.ResumeID,
				TopK:       &topK,
			})
			if err != nil {
				return fmt.Sprintf("检索简历异常: %v", err), nil
			}

			return formatResp(resp), nil
		})
}

// 2. 封装“查 JD”工具
type JDSkillInput struct {
	Query   string `json:"query" jsonschema:"description=要查询的岗位能力要求"`
	Company string `json:"company" jsonschema:"description=公司名称，选填"`
}

func NewSearchJDTool(svc *svc.ServiceContext) (tool.BaseTool, error) {
	return utils.InferTool("SearchJDSkill", "检索该岗位(JD)的核心要求，用于对标提问",
		func(ctx context.Context, input *JDSkillInput) (string, error) {
			topK := int32(3)
			resp, err := retrieve.NewRetrieveService(svc).RetrieveContext(ctx, &agent.RetrieveReq{
				Query:      input.Query,
				Collection: "goffer_jd_bank",
				Company:    &input.Company,
				TopK:       &topK,
			})
			if err != nil {
				return fmt.Sprintf("检索jd异常: %v", err), nil
			}
			return formatResp(resp), nil
		})
}

// 3. 封装“查题库”工具
type QuestionBankInput struct {
	Query string `json:"query" jsonschema:"description=要查询的技术知识点或面试题，例如'Redis缓存击穿','GMP模型'"`
	// 可选参数：如果你的 RAG 服务支持根据难度过滤，大模型可以传这个字段
	Difficulty string `json:"difficulty,omitempty" jsonschema:"description=题目难度，选填，如'简单','中等','困难'"`
}

func NewSearchQuestionBankTool(svc *svc.ServiceContext) (tool.BaseTool, error) {
	return utils.InferTool("SearchQuestionBankSkill", "从标准面试题库中检索专业知识点的标准答案和考察维度",
		func(ctx context.Context, input *QuestionBankInput) (string, error) {
			topK := int32(2) // 题库答案一般比较长，取 Top 2 就够了

			// 构造请求
			req := &agent.RetrieveReq{
				Query:      input.Query,
				Collection: "goffer_question_bank", // 🌟 锁定题库集合
				TopK:       &topK,
			}

			// 如果大模型指定了难度（前提是你的 IDL 里加了 Difficulty 字段）
			// if input.Difficulty != "" { req.Difficulty = &input.Difficulty }

			resp, err := retrieve.NewRetrieveService(svc).RetrieveContext(ctx, req)
			if err != nil {
				return fmt.Sprintf("检索题库异常: %v", err), nil
			}

			return formatResp(resp), nil
		})
}

// formatResp 将检索回来的多个上下文切片拼接成一个长文本
func formatResp(resp *agent.RetrieveResp) string {
	if resp == nil || len(resp.Contexts) == 0 {
		return "（未检索到相关的参考信息，请基于自身知识回答或追问细节）"
	}
	// 用分隔符把多个片段拼起来，大模型对 "---" 这种分隔符的理解能力很好
	return strings.Join(resp.Contexts, "\n---\n")
}
