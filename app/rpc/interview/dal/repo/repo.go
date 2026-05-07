package repo

import (
	"Goffer/app/rpc/interview/dal/mongodb"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ChatMessage 代表单条聊天消息
type ChatMessage struct {
	Role    string
	Content string
}

// ChatContext 代表获取到的上下文聚合数据
type ChatContext struct {
	FsmState  string
	History   []*ChatMessage
	RagChunks string
}

func (s *RepoService) GetChatContextInterview(ctx context.Context, SessionId, LatestUserMsg string) (*ChatContext, error) {
	// 1. 从 Redis 获取当前环节状态
	fsmKey := fmt.Sprintf("interview:fsm:%s", SessionId)
	fsmStr, err := s.Cache.Get(ctx, fsmKey).Result()
	if err != nil {
		return nil, fmt.Errorf("从 Redis 获取状态机失败: %w", err)
	}

	// 从 Redis 读出来的是 JSON 字符串，需要反序列化 (Unmarshal) 为 map 或 struct
	var fsmState map[string]interface{}
	if err := json.Unmarshal([]byte(fsmStr), &fsmState); err != nil {
		return nil, fmt.Errorf("解析 FSM 状态失败: %w", err)
	}

	// 2. 从 MongoDB 拉取最近 5 轮历史对话
	history, err := s.Mongo.GetRecentChatHistory(ctx, SessionId, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chat history from MongoDB: %w", err)
	}

	// 3. 用 req.LatestUserMsg 去 Qdrant 进行向量检索
	ragChunks, err := s.VectorStore.Search(ctx, LatestUserMsg)
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
	respMessages := make([]*ChatMessage, 0, len(history))
	for _, h := range history {
		respMessages = append(respMessages, &ChatMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	// 提取当前的状态和轮次，给提示词工程使用
	currentStatus := ""
	if status, ok := fsmState["status"].(string); ok {
		currentStatus = status
	}

	return &ChatContext{
		FsmState:  currentStatus, // 例如: "greeting", "project_deep_dive"
		History:   respMessages,  // 组装好的最近 5 轮对话数组
		RagChunks: resumeContext, // 从 Qdrant 捞出来的简历文本片段
	}, nil
}

func (s *RepoService) SaveChatRecordInterview(ctx context.Context, sessionID, UserMsg, AiMsg, NextState string) error {
	// 1. 保存用户消息到 MongoDB
	userMsg := mongodb.Message{
		Role:    "user",
		Content: UserMsg,
		Time:    time.Now().Unix(),
	}
	if err := s.Mongo.AppendMessage(ctx, sessionID, userMsg); err != nil {
		return fmt.Errorf("保存用户聊天记录失败: %w", err)
	}

	// 2. 保存 AI 消息到 MongoDB
	aiMsg := mongodb.Message{
		Role:    "assistant",
		Content: AiMsg,
		Time:    time.Now().Unix(),
	}
	if err := s.Mongo.AppendMessage(ctx, sessionID, aiMsg); err != nil {
		return fmt.Errorf("保存面试官聊天记录失败: %w", err)
	}

	// 3. (可选) 更新 Redis 的状态机，决定是否进入下一环节
	if NextState != "" {
		fsmKey := fmt.Sprintf("interview:fsm:%s", sessionID)

		// 3.1 最佳实践：先读取老状态，防止覆盖掉 "round" (轮次) 等其他无关数据
		// 注意：如果使用 go-redis，Get 返回的 value 通常用 .Result() 获取字符串
		fsmStr, err := s.Cache.Get(ctx, fsmKey).Result()
		if err == nil && fsmStr != "" {
			var fsmState map[string]interface{}
			if err := json.Unmarshal([]byte(fsmStr), &fsmState); err == nil {
				// 3.2 更新状态
				fsmState["status"] = NextState

				// 3.3 (可选) 每次保存一轮完整对话，将对话轮次 + 1
				// 注意：Go 中 json.Unmarshal 默认将数字解析为 float64
				if round, ok := fsmState["round"].(float64); ok {
					fsmState["round"] = round + 1
				}

				// 3.4 重新写回 Redis，并刷新过期时间（例如 2 小时）
				fsmBytes, _ := json.Marshal(fsmState)
				err = s.Cache.Set(ctx, fsmKey, fsmBytes, 2*time.Hour).Err()
				if err != nil {
					// 记录日志，但不阻断流程，因为聊天记录已经保存成功
					fmt.Printf("Warning: Failed to update FSM state in Redis: %v\n", err)
				}
			}
		}
	}

	return nil
}
