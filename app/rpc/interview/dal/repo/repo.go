package repo

import (
	"Goffer/app/rpc/interview/dal/mongodb"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type FSMState struct {
	Status   string `json:"status"`
	Round    int    `json:"round"`
	ResumeID string `json:"resume_id"`
}

// ChatMessage 代表单条聊天消息
type ChatMessage struct {
	Role    string
	Content string
}

// ChatContext 代表获取到的上下文聚合数据
type ChatContext struct {
	FsmState string
	History  []*ChatMessage
	ResumeId string
}

func (s *RepoService) GetChatContextInterview(ctx context.Context, SessionId string) (*ChatContext, error) {
	// 1. 从 Redis 获取当前环节状态
	fsmKey := fmt.Sprintf("interview:fsm:%s", SessionId)
	fsmStr, err := s.Cache.Get(ctx, fsmKey).Result()
	if err != nil {
		return nil, fmt.Errorf("从 Redis 获取状态机失败: %w", err)
	}

	// 从 Redis 读出来的是 JSON 字符串，需要反序列化 (Unmarshal) 为 map 或 struct
	var fsmState FSMState
	if err := json.Unmarshal([]byte(fsmStr), &fsmState); err != nil {
		return nil, fmt.Errorf("解析 FSM 状态失败: %w", err)
	}

	// 2. 从 MongoDB 拉取最近 5 轮历史对话
	history, err := s.Mongo.GetRecentChatHistory(ctx, SessionId, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat history from MongoDB: %w", err)
	}

	// 5. 组装并返回上下文给外层
	// 注意：这里的 interview.Message 假设是你 proto 文件中定义的 message 结构
	respMessages := make([]*ChatMessage, 0, len(history))
	for _, h := range history {
		respMessages = append(respMessages, &ChatMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	return &ChatContext{
		FsmState: fsmState.Status, // 例如: "greeting", "project_deep_dive"
		History:  respMessages,    // 组装好的最近 5 轮对话数组
		ResumeId: fsmState.ResumeID,
	}, nil
}

func (s *RepoService) SaveChatRecordInterview(ctx context.Context, sessionID, UserMsg, AiMsg string) error {
	userMsg := mongodb.Message{
		Role:    "user",
		Content: UserMsg,
		Time:    time.Now().Unix(),
	}
	if err := s.Mongo.AppendMessage(ctx, sessionID, userMsg); err != nil {
		return fmt.Errorf("保存用户聊天记录失败: %w", err)
	}

	aiMsg := mongodb.Message{
		Role:    "assistant",
		Content: AiMsg,
		Time:    time.Now().Unix(),
	}
	if err := s.Mongo.AppendMessage(ctx, sessionID, aiMsg); err != nil {
		return fmt.Errorf("保存面试官聊天记录失败: %w", err)
	}

	return nil
}
