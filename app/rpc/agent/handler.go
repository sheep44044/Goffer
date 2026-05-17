package agent

import (
	"Goffer/app/rpc/agent/bot"
	"Goffer/app/rpc/agent/rag/retrieve"
	"Goffer/app/rpc/agent/svc"
	"Goffer/app/rpc/user/pack"
	"Goffer/kitex_gen/agent"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
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

	logger.InfoCtx(ctx, "RPC 检索请求", zap.String("user_id", req.UserId), zap.String("collection", req.Collection))

	retData, err := retrieve.NewRetrieveService(s.svc).RetrieveContext(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "检索失败", zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	if retData != nil {
		resp.Contexts = retData.Contexts
	}

	return resp, nil
}

func (s *AgentServiceImpl) ChatStream(req *agent.ChatStreamReq, stream agent.AgentService_ChatStreamServer) error {
	// 使用 stream.Context() 继承上游 OTel TraceID，而非 context.Background()
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	roomID := req.SessionId // SessionId 即面试房间 ID
	s.svc.CancelManager.Register(roomID, cancel)
	defer s.svc.CancelManager.Remove(roomID)

	// 2. 将 Thrift 请求转换为 Eino DAG 需要的输入格式
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

	// 3. 使用可取消的 ctx 调用 Eino DAG
	botName := getBotNameByFsmState(req.FsmState)
	aiStream, err := bot.GetBotManager().StreamAnswer(ctx, botName, input)
	if err != nil {
		return fmt.Errorf("Agent 思考失败: %w", err)
	}
	defer aiStream.Close()

	// 4. 循环接收 Eino 生成的文字，并通过流式接口发送给 Interview 服务
	for {
		msg, err := aiStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				logger.InfoCtx(ctx, "Agent 推理被用户打断", zap.String("room", roomID))
				return nil
			}
			return err
		}

		if msg.Content != "" {
			err = stream.Send(&agent.ChatStreamResp{
				Chunk: msg.Content,
			})
			if err != nil {
				logger.ErrorCtx(ctx, "流式推送失败(Interview 可能断开)", zap.String("room", roomID), zap.Error(err))
				return nil
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
