package main

import (
	"context"
	"encoding/json"

	"Goffer/app/rpc/earmouth/pipeline"
	"Goffer/app/rpc/earmouth/svc"
	"Goffer/pkg/logger"

	"go.uber.org/zap"
)

const cancelChannel = "interview:cancel_events"

type cancelEvent struct {
	RoomID    string `json:"room_id"`
	Timestamp int64  `json:"timestamp"`
}

type EarMouthServiceImpl struct {
	svc      *svc.ServiceContext
	inbound  *pipeline.InboundPipeline
	outbound *pipeline.OutboundPipeline
}

func NewEarMouthService(svc *svc.ServiceContext) *EarMouthServiceImpl {
	return &EarMouthServiceImpl{
		svc:      svc,
		inbound:  pipeline.NewInboundPipeline(svc),
		outbound: pipeline.NewOutboundPipeline(svc),
	}
}

func (s *EarMouthServiceImpl) Start() {
	go s.inbound.Start()
	go s.outbound.Start()
	s.subscribeCancelEvents()

	logger.Info("Inbound 管道已启动: audio.in -> STT -> text.in")
	logger.Info("Outbound 管道已启动: text.out -> TTS -> audio.out (含打断检测)")
}

func (s *EarMouthServiceImpl) subscribeCancelEvents() {
	go func() {
		ctx := context.Background()
		pubsub := s.svc.RedisClient.Subscribe(ctx, cancelChannel)
		defer pubsub.Close()

		if _, err := pubsub.Receive(ctx); err != nil {
			logger.Error("Redis 订阅打断事件失败", zap.Error(err))
			return
		}
		logger.Info("已订阅 Redis Channel", zap.String("channel", cancelChannel))

		ch := pubsub.Channel()
		for msg := range ch {
			var event cancelEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				logger.Warn("解析打断事件失败", zap.Error(err))
				continue
			}
			if event.RoomID == "" {
				continue
			}
			s.svc.CancelTracker.Record(event.RoomID, event.Timestamp)
			logger.Info("记录打断时间戳", zap.String("room", event.RoomID), zap.Int64("ts", event.Timestamp))
		}
		logger.Info("Redis 订阅通道已关闭")
	}()
}
