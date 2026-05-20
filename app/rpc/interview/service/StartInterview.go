package service

import (
	"Goffer/app/rpc/interview/dal/repo"
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/interview"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"Goffer/pkg/snowflake"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type StartService struct {
	svc *svc.ServiceContext
}

func NewStartService(svc *svc.ServiceContext) *StartService {
	return &StartService{
		svc: svc,
	}
}

func (s *StartService) StartInterview(ctx context.Context, req *interview.StartInterviewReq) (*interview.StartInterviewResp, error) {
	checkResp, err := s.svc.UserClient.CheckResumeStatus(ctx, &user.CheckResumeStatusReq{
		UserId:   req.UserId,
		ResumeId: req.ResumeId,
	})
	if err != nil {
		return nil, fmt.Errorf("调用 User 服务查询简历失败: %w", err)
	}

	if checkResp.ParseStatus != 2 {
		// 返回业务错误：简历未就绪
		return nil, errno.ServiceErr.WithMessage("您的简历还在被 AI 努力阅读中，请稍后再试")
	}

	sessionID := snowflake.GenString()

	fsmKey := fmt.Sprintf("interview:fsm:%s", sessionID)
	fsmState := repo.FSMState{
		Status:   "greeting", // 初始状态为打招呼
		Round:    0,          // 对话轮次初始化为 0
		ResumeID: req.ResumeId,
	}
	fsmBytes, err := json.Marshal(fsmState)
	if err != nil {
		return nil, fmt.Errorf("序列化 FSM 初始状态失败: %w", err)
	}

	// 将状态写入 Redis，并设置一个过期时间 (比如 2 小时后自动销毁房间)
	err = s.svc.Cache.Set(ctx, fsmKey, fsmBytes, 2*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("初始化 Redis 状态机失败: %w", err)
	}

	chatHistoryColl := s.svc.Mongo.DB.Collection("chat_history")
	initialDoc := bson.M{
		"session_id": sessionID,
		"user_id":    req.UserId,
		"resume_id":  req.ResumeId,
		"start_time": time.Now().Unix(),
		"messages":   []bson.M{}, // 空的聊天记录数组
	}

	_, err = chatHistoryColl.InsertOne(ctx, initialDoc)
	if err != nil {
		// 严谨的做法：如果 Mongo 写入失败，最好把刚写进 Redis 的状态也删掉 (补偿操作)
		s.svc.Cache.Del(ctx, fsmKey)
		return nil, fmt.Errorf("初始化 MongoDB 记忆失败: %w", err)
	}

	openingRemark := "你好，我是本次的面试官。我已经仔细阅读了你的简历，请先做一个简单的两分钟自我介绍吧。"

	return &interview.StartInterviewResp{
		SessionId:     sessionID,
		OpeningRemark: openingRemark,
	}, nil
}
