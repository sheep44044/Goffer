package agent

import (
	"Goffer/app/rpc/agent/bot"
	"Goffer/app/rpc/agent/rag/retrieve"
	"Goffer/app/rpc/agent/svc"
	"Goffer/app/rpc/user/pack"
	"Goffer/kitex_gen/agent"
	"Goffer/pkg/errno"
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
)

type AgentServiceImpl struct {
	svc *svc.ServiceContext
}

func NewAgentService(svc *svc.ServiceContext) *AgentServiceImpl {
	return &AgentServiceImpl{svc: svc}
}

func (s *AgentServiceImpl) RetrieveContext(ctx context.Context, req *agent.RetrieveReq) (resp *agent.RetrieveResp, err error) {
	resp = new(agent.RetrieveResp)

	if len(req.UserId) == 0 || len(req.Collection) == 0 || len(req.Query) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	retData, err := retrieve.NewRetrieveService(s.svc).RetrieveContext(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	if retData != nil {
		resp.Contexts = retData.Contexts
	}

	return resp, nil
}

func (s *AgentServiceImpl) ChatStream(req *agent.ChatStreamReq, stream agent.AgentService_ChatStreamServer) error {
	// 1. 将 Thrift 请求转换为 Eino DAG 需要的输入格式
	ctx := context.Background()
	var einoHistory []*schema.Message
	for _, msg := range req.History {
		if msg.Role == "user" {
			einoHistory = append(einoHistory, schema.UserMessage(msg.Content))
		} else {
			einoHistory = append(einoHistory, schema.AssistantMessage(msg.Content, nil))
		}
	}

	input := bot.BotInput{
		SessionID: req.SessionId,
		UserID:    req.UserId,
		Message:   req.Message,
		FsmState:  req.FsmState,
		ResumeID:  req.ResumeId,
		History:   einoHistory,
	}

	// 2. 调用我们在 bot 模块写好的管理器
	botName := getBotNameByFsmState(req.FsmState)
	aiStream, err := bot.GetBotManager().StreamAnswer(ctx, botName, input)
	if err != nil {
		return fmt.Errorf("Agent 思考失败: %v", err)
	}
	defer aiStream.Close()

	// 3. 循环接收 Eino 生成的文字，并通过流式接口发送给 Interview 服务
	for {
		msg, err := aiStream.Recv()
		if err == io.EOF {
			break // AI 说完了
		}
		if err != nil {
			return err
		}

		if msg.Content != "" {
			// 把切片推回给 Interview 客户端
			err = stream.Send(&agent.ChatStreamResp{
				Chunk: msg.Content,
			})
			if err != nil {
				return err // 如果推送失败（比如客户端断开了），直接退出
			}
		}
	}

	return nil
}

func getBotNameByFsmState(fsmState string) string {
	switch fsmState {
	case "greeting":
		return "HR_Interviewer"
	case "tech_foundation":
		return "Foundation_Interviewer"
	case "tech_architecture":
		return "Architecture_Interviewer"
	case "evaluator":
		return "Interview_Evaluator"
	default:
		return "HR_Interviewer"
	}
}
