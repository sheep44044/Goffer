package service

import (
	"Goffer/app/rpc/interview/dal/repo"
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/interview"
	"context"
	"encoding/json"
	"fmt"
)

type ResumeSessionService struct {
	svc *svc.ServiceContext
}

func NewResumeSessionService(svc *svc.ServiceContext) *ResumeSessionService {
	return &ResumeSessionService{svc: svc}
}

func (s *ResumeSessionService) ResumeSession(ctx context.Context, req *interview.ResumeSessionReq) (*interview.ResumeSessionResp, error) {
	fsmKey := fmt.Sprintf("interview:fsm:%s", req.SessionId)
	fsmStr, err := s.svc.Cache.Get(ctx, fsmKey).Result()
	if err != nil {
		return nil, fmt.Errorf("session 不存在或已过期: %w", err)
	}

	var fsmState repo.FSMState
	if err := json.Unmarshal([]byte(fsmStr), &fsmState); err != nil {
		return nil, fmt.Errorf("解析 FSM 状态失败: %w", err)
	}

	history, err := s.svc.Mongo.GetRecentChatHistory(ctx, req.SessionId, 50)
	if err != nil {
		return nil, fmt.Errorf("获取聊天记录失败: %w", err)
	}

	respMessages := make([]*interview.ChatMessage, 0, len(history))
	for _, h := range history {
		respMessages = append(respMessages, &interview.ChatMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	return &interview.ResumeSessionResp{
		FsmState: fsmState.Status,
		Round:    int32(fsmState.Round),
		History:  respMessages,
	}, nil
}
