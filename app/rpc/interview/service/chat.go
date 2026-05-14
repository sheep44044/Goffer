package service

import (
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/agent" // 引入你刚生成的 agent thrift 代码
	"Goffer/kitex_gen/interview"
	"Goffer/pkg/contextutil"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"
)

type ChatService struct {
	svc *svc.ServiceContext
}

func NewChatService(svc *svc.ServiceContext) *ChatService {
	return &ChatService{svc: svc}
}

func (s *ChatService) ChatStream(ctx context.Context, req *interview.ChatReq, stream interview.InterviewService_ChatStreamServer) error {
	err := s.advanceFSM(ctx, req.SessionId)
	if err != nil {
		log.Printf("⚠️ 推进状态机失败，但不阻断聊天: %v", err)
	}
	// 1. 获取业务情报 (查 DB/Redis)
	// ⚠️ 注意：这里的 contextInfo 里现在应该包含 ResumeId 了（见下方的修改）
	contextInfo, err := s.svc.Repo.GetChatContextInterview(ctx, req.SessionId)
	if err != nil {
		return fmt.Errorf("获取上下文失败: %w", err)
	}

	userID, _ := contextutil.GetUserIDFromRPC(ctx)

	// 🌟 核心修复 1：解决 Thrift 类型不匹配问题 (手动映射)
	var agentHistory []*agent.Message
	for _, h := range contextInfo.History {
		agentHistory = append(agentHistory, &agent.Message{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	// ==========================================
	// 🌟 核心修复 2：组装完整的 Agent 请求参数
	// ==========================================
	agentReq := &agent.ChatStreamReq{
		SessionId: req.SessionId,
		UserId:    userID,
		Message:   req.Message,
		FsmState:  contextInfo.FsmState,
		ResumeId:  contextInfo.ResumeId, // 🌟 从后台上下文拿，而不是前端传！
		History:   agentHistory,         // 🌟 传入转换后的类型
	}

	// 3. 呼叫 Agent 服务，开启 AI 思考流 (RPC Stream)
	agentStream, err := s.svc.AgentStreamClient.ChatStream(ctx, agentReq)
	if err != nil {
		return fmt.Errorf("调用 Agent 服务失败: %w", err)
	}

	fullAnswer := ""

	// 4. 循环接收 Agent 的流，并无缝转发给 Gateway
	for {
		agentResp, err := agentStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 Agent 流异常: %w", err)
		}

		if agentResp.Chunk != "" {
			fullAnswer += agentResp.Chunk
			err = stream.Send(&interview.ChatResp{
				Chunk: agentResp.Chunk,
			})
			if err != nil {
				log.Printf("网关断开连接，停止发送: %v", err)
				break
			}
		}
	}

	// 5. 异步落地“战后记忆”
	go func(sid, userMsg, aiMsg string) {
		bgCtx := context.Background()
		_ = s.svc.Repo.SaveChatRecordInterview(bgCtx, sid, userMsg, aiMsg, "2")
	}(req.SessionId, req.Message, fullAnswer)

	return nil
}

func (s *ChatService) advanceFSM(ctx context.Context, sessionID string) error {
	fsmKey := fmt.Sprintf("interview:fsm:%s", sessionID)

	// 1. 获取当前状态
	fsmStr, err := s.svc.Cache.Get(ctx, fsmKey).Result()
	if err != nil {
		return fmt.Errorf("读取状态机失败: %w", err)
	}

	var fsmState map[string]interface{}
	if err := json.Unmarshal([]byte(fsmStr), &fsmState); err != nil {
		return fmt.Errorf("解析状态机失败: %w", err)
	}

	// 2. 推进逻辑
	currentStatus := fsmState["status"].(string)
	round := int(fsmState["round"].(float64))

	round++
	nextStatus := currentStatus

	switch currentStatus {
	case "greeting":
		if round >= 1 {
			nextStatus = "tech_foundation"
			round = 0
		}
	case "tech_foundation":
		if round >= 4 {
			nextStatus = "tech_architecture"
			round = 0
		}
	case "tech_architecture":
		if round >= 4 {
			nextStatus = "evaluator"
			round = 0
		}
	case "evaluator":
		return nil // 已经是最终状态，不推进，直接返回
	}

	// 3. 构造新状态并写回 Redis
	fsmState["status"] = nextStatus
	fsmState["round"] = round
	// resume_id 天然还在 fsmState 里，不需要额外处理！

	fsmBytes, _ := json.Marshal(fsmState)
	return s.svc.Cache.Set(ctx, fsmKey, fsmBytes, 2*time.Hour).Err()
}
