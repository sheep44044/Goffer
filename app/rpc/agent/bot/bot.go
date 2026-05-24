package bot

import (
	"Goffer/kitex_gen/agent"
	"context"
	"fmt"
	"strings"

	"Goffer/app/rpc/agent/presets"
	"Goffer/app/rpc/agent/rag/retrieve"
	"Goffer/app/rpc/agent/svc"
	"Goffer/app/rpc/agent/tools"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// BotInput 组装给 Eino 图的输入
type BotInput struct {
	SessionID string
	UserID    string
	Message   string
	FsmState  string
	History   []*schema.Message
	ResumeID  string // 用于 RAG 过滤
}

// InterviewBot 封装编译好的执行器
type InterviewBot struct {
	Name      string
	dagRunner compose.Runnable[BotInput, *schema.Message]
}

// NewInterviewBot 核心组装逻辑：将 RAG、MCP 和 Prompt 编排进 DAG 图
func NewInterviewBot(preset *presets.InterviewerPreset, svc *svc.ServiceContext) (*InterviewBot, error) {
	// 1. 初始化执行链 (Chain)
	chain := compose.NewChain[BotInput, *schema.Message]()

	// 节点 1：动态并行检索层 (Dynamic Parallel)
	parallel := compose.NewParallel()

	// 数据透传 (无论什么面试官，都需要透传基础数据)
	parallel.AddLambda("raw_input", compose.InvokableLambda(func(ctx context.Context, input BotInput) (BotInput, error) {
		return input, nil
	}))

	// 动态判断 1：哪些面试官需要看简历？(HR、基础技术、架构技术都需要，但打分专家可能不需要，因为他只看历史记录)
	needsResume := preset.Name == "HR_Interviewer" || preset.Name == "Foundation_Interviewer" || preset.Name == "Architecture_Interviewer"
	if needsResume {
		parallel.AddLambda("resume_context", compose.InvokableLambda(func(ctx context.Context, input BotInput) (string, error) {
			topK := int32(3)
			resp, err := retrieve.NewRetrieveService(svc).RetrieveContext(ctx, &agent.RetrieveReq{
				Query:      input.Message,
				UserId:     input.UserID,
				Collection: "goffer_resumes",
				ResumeId:   &input.ResumeID,
				TopK:       &topK,
			})
			if err != nil || len(resp.Contexts) == 0 {
				return "（未搜寻到相关简历背景）", nil
			}
			return strings.Join(resp.Contexts, "\n---\n"), nil
		}))
	}

	// 动态判断 2：哪些面试官需要看八股文题库？(主要是基础技术面试官)
	needsQB := preset.Name == "Foundation_Interviewer"
	if needsQB {
		parallel.AddLambda("qb_context", compose.InvokableLambda(func(ctx context.Context, input BotInput) (string, error) {
			topK := int32(2)
			resp, err := retrieve.NewRetrieveService(svc).RetrieveContext(ctx, &agent.RetrieveReq{
				Query:      input.Message,
				UserId:     input.UserID,
				Collection: "goffer_question_bank",
				TopK:       &topK,
			})
			if err != nil || len(resp.Contexts) == 0 {
				return "（未搜寻到相关标准题库知识）", nil
			}
			return strings.Join(resp.Contexts, "\n---\n"), nil
		}))
	}

	chain.AppendParallel(parallel)

	// 节点 2：Prompt 逻辑组装层 (Lambda)
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, pMap map[string]any) ([]*schema.Message, error) {
		input, ok := pMap["raw_input"].(BotInput)
		if !ok {
			return nil, fmt.Errorf("DAG内部错误: raw_input 类型不匹配")
		}

		// 安全提取：因为有些分支根据人设没有被挂载，直接使用 .(string) 会导致 panic
		resumeTxt := "（当前面试官无需参考简历）"
		if val, ok := pMap["resume_context"]; ok {
			resumeTxt = val.(string)
		}

		qbTxt := "（当前面试官无需参考题库）"
		if val, ok := pMap["qb_context"]; ok {
			qbTxt = val.(string)
		}

		// 将 SessionID 作为隐式系统参数
		fullSystemPrompt := fmt.Sprintf(`%s

# 内部系统变量（绝密：仅用于工具调用，请勿向用户暴露）
- 当前面试 SessionID : %s
- 当前面试环节状态 : %s

# 面试上下文辅助信息
【候选人简历相关内容】：
%s

【面试题目参考知识】：
%s

请严格遵循你的系统人设。结合以上信息与候选人交谈（或进行客观打分评估）。`,
			preset.SystemPrompt,
			input.SessionID,
			input.FsmState,
			resumeTxt,
			qbTxt,
		)

		// 组装最终消息序列
		msgs := []*schema.Message{
			schema.SystemMessage(fullSystemPrompt),
		}
		msgs = append(msgs, input.History...)
		msgs = append(msgs, schema.UserMessage(input.Message))

		return msgs, nil
	}))

	// 节点 3：核心推理层 (Agent or ChatModel)
	activeTools := tools.GetToolsByName(preset.AllowedTools)

	if len(activeTools) > 0 {
		// 如果有动作型工具，封装为 ReAct Agent
		agent, err := react.NewAgent(context.Background(), &react.AgentConfig{
			ToolCallingModel: svc.EinoChatModel,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: activeTools,
			},
			MaxStep: 5,
		})
		if err != nil {
			return nil, fmt.Errorf("创建 Agent 节点失败: %w", err)
		}
		subGraph, opts := agent.ExportGraph()
		chain.AppendGraph(subGraph, opts...)
	} else {
		// 如果没有配置工具，直接挂载原生大模型，速度最快
		chain.AppendChatModel(svc.EinoChatModel)
	}

	// 4. 编译 DAG 图
	runner, err := chain.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("编译 Eino DAG 图失败: %w", err)
	}

	return &InterviewBot{
		Name:      preset.Name,
		dagRunner: runner,
	}, nil
}

// StreamAnswer 对外暴露流式对话接口
func (b *InterviewBot) StreamAnswer(ctx context.Context, input BotInput) (*schema.StreamReader[*schema.Message], error) {
	return b.dagRunner.Stream(ctx, input)
}
