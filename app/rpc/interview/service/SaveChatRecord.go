package service

import (
	"Goffer/app/rpc/interview/dal/mongodb"
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/interview"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type SaveChatService struct {
	svc *svc.ServiceContext
}

func NewSaveChatService(svc *svc.ServiceContext) *SaveChatService {
	return &SaveChatService{
		svc: svc,
	}
}

func (s *SaveChatService) SaveChatRecordInterview(ctx context.Context, req *interview.SaveChatRecordReq) error {
	// 1. 保存用户消息到 MongoDB
	userMsg := mongodb.Message{
		Role:    "user",
		Content: req.UserMsg,
		Time:    time.Now().Unix(),
	}
	if err := s.svc.Mongo.AppendMessage(ctx, req.SessionId, userMsg); err != nil {
		return fmt.Errorf("保存用户聊天记录失败: %w", err)
	}

	// 2. 保存 AI 消息到 MongoDB
	aiMsg := mongodb.Message{
		Role:    "assistant",
		Content: req.AiMsg,
		Time:    time.Now().Unix(),
	}
	if err := s.svc.Mongo.AppendMessage(ctx, req.SessionId, aiMsg); err != nil {
		return fmt.Errorf("保存面试官聊天记录失败: %w", err)
	}

	// 3. (可选) 更新 Redis 的状态机，决定是否进入下一环节
	if req.NextState != "" {
		fsmKey := fmt.Sprintf("interview:fsm:%s", req.SessionId)

		// 3.1 最佳实践：先读取老状态，防止覆盖掉 "round" (轮次) 等其他无关数据
		// 注意：如果使用 go-redis，Get 返回的 value 通常用 .Result() 获取字符串
		fsmStr, err := s.svc.Cache.Get(ctx, fsmKey).Result()
		if err == nil && fsmStr != "" {
			var fsmState map[string]interface{}
			if err := json.Unmarshal([]byte(fsmStr), &fsmState); err == nil {
				// 3.2 更新状态
				fsmState["status"] = req.NextState

				// 3.3 (可选) 每次保存一轮完整对话，将对话轮次 + 1
				// 注意：Go 中 json.Unmarshal 默认将数字解析为 float64
				if round, ok := fsmState["round"].(float64); ok {
					fsmState["round"] = round + 1
				}

				// 3.4 重新写回 Redis，并刷新过期时间（例如 2 小时）
				fsmBytes, _ := json.Marshal(fsmState)
				err = s.svc.Cache.Set(ctx, fsmKey, fsmBytes, 2*time.Hour).Err()
				if err != nil {
					// 记录日志，但不阻断流程，因为聊天记录已经保存成功
					fmt.Printf("Warning: Failed to update FSM state in Redis: %v\n", err)
				}
			}
		}
	}

	return nil
}
