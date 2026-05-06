package service

import (
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/interview"
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
	var fsmState map[string]interface{}
	if err := json.Unmarshal([]byte(fsmStr), &fsmState); err != nil {
		return nil, fmt.Errorf("解析 FSM 状态失败: %w", err)
	}

	// 2. 从 MongoDB 拉取最近 5 轮历史对话
	history, err := s.svc.Mongo.GetRecentChatHistory(ctx, req.SessionId, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat history from MongoDB: %w", err)
	}

	// 3. 用 req.LatestUserMsg 去 Qdrant 进行向量检索
	ragChunks, err := s.svc.VectorStore.Search(ctx, req.LatestUserMsg)
	if err != nil {
		return nil, err
	}

	// 4. 将 Qdrant 检索到的 Document 组装成纯文本上下文
	var resumeContext string
	for i, chunk := range ragChunks {
		// 根据你的 chunk 结构拼接，提供给大模型参考
		resumeContext += fmt.Sprintf("[片段%d] %s\n", i+1, chunk.Content)
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

	// 提取当前的状态和轮次，给提示词工程使用
	currentStatus := ""
	if status, ok := fsmState["status"].(string); ok {
		currentStatus = status
	}

	return &interview.GetChatContextResp{
		FsmState:  currentStatus, // 例如: "greeting", "project_deep_dive"
		History:   respMessages,  // 组装好的最近 5 轮对话数组
		RagChunks: resumeContext, // 从 Qdrant 捞出来的简历文本片段
	}, nil
}
