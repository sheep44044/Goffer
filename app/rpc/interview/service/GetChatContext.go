package service

import (
	"Goffer/app/rpc/interview/dal/repo"
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/agent"
	"Goffer/kitex_gen/interview"
	"Goffer/pkg/contextutil"
	"context"
	"encoding/json"
	"fmt"
)

type GetChatService struct {
	svc *svc.ServiceContext
}

func NewGetChatService(svc *svc.ServiceContext) *GetChatService {
	return &GetChatService{
		svc: svc,
	}
}

func (s *GetChatService) GetChatContextInterview(ctx context.Context, req *interview.GetChatContextReq) (*interview.GetChatContextResp, error) {
	// 1. 从 Redis 获取当前环节状态
	fsmKey := fmt.Sprintf("interview:fsm:%s", req.SessionId)
	fsmStr, err := s.svc.Cache.Get(ctx, fsmKey).Result()
	if err != nil {
		return nil, fmt.Errorf("从 Redis 获取状态机失败: %w", err)
	}

	// 从 Redis 读出来的是 JSON 字符串，需要反序列化 (Unmarshal) 为 map 或 struct
	var fsmState repo.FSMState
	if err := json.Unmarshal([]byte(fsmStr), &fsmState); err != nil {
		return nil, fmt.Errorf("解析 FSM 状态失败: %w", err)
	}

	// 2. 从 MongoDB 拉取最近 5 轮历史对话
	history, err := s.svc.Mongo.GetRecentChatHistory(ctx, req.SessionId, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat history from MongoDB: %w", err)
	}

	userID, _ := contextutil.GetUserIDFromRPC(ctx)

	topK := int32(3)
	ragResp, err := s.svc.AgentClient.RetrieveContext(ctx, &agent.RetrieveReq{
		Query:      req.LatestUserMsg,
		UserId:     userID,
		ResumeId:   &req.ResumeId,
		Collection: "goffer_resumes",
		TopK:       &topK,
	})
	if err != nil {
		return nil, fmt.Errorf("RAG检索失败: %w", err)
	}

	// 4. 将 Qdrant 检索到的结果组装成纯文本上下文
	var resumeContext string
	if ragResp != nil && len(ragResp.Contexts) > 0 {
		for i, text := range ragResp.Contexts {
			resumeContext += fmt.Sprintf("[片段%d] %s\n", i+1, text)
		}
	} else {
		resumeContext = "（未检索到简历相关信息）"
	}

	// 5. 组装并返回上下文给外层
	// 注意：这里的 interview.Message 假设是你 proto 文件中定义的 message 结构
	respMessages := make([]*interview.ChatMessage, 0, len(history))
	for _, h := range history {
		respMessages = append(respMessages, &interview.ChatMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	return &interview.GetChatContextResp{
		FsmState:  fsmState.Status, // 例如: "greeting", "project_deep_dive"
		History:   respMessages,    // 组装好的最近 5 轮对话数组
		RagChunks: resumeContext,   // 从 Qdrant 捞出来的简历文本片段
	}, nil
}
