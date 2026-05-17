package pipeline

import (
	"Goffer/app/rpc/earmouth/mq"
	"Goffer/app/rpc/earmouth/svc"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type AudioFrame struct {
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
	Codec     string `json:"codec"`
	Data      []byte `json:"data"`
}

type InboundPipeline struct {
	svc          *svc.ServiceContext
	roomSessions sync.Map
}

type roomSession struct {
	roomID    string
	userID    string
	audioCh   chan []byte
	pipeline  *InboundPipeline
	startOnce sync.Once
	timer     *time.Timer
	mu        sync.Mutex
}

func NewInboundPipeline(svc *svc.ServiceContext) *InboundPipeline {
	return &InboundPipeline{svc: svc}
}

func (p *InboundPipeline) Start() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        p.svc.Config.Kafka.Brokers,
		GroupID:        p.svc.Config.Kafka.ConsumerGroup + "-inbound",
		Topic:          p.svc.Config.Kafka.AudioInTopic,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second,
		MinBytes:       1,
		MaxBytes:       10e6,
	})
	defer reader.Close()

	ctx := context.Background()
	logger.Info("开始消费 audio.in topic", zap.String("topic", p.svc.Config.Kafka.AudioInTopic))

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			logger.Warn("拉取 Kafka 消息失败", zap.Error(err))
			continue
		}

		var frame AudioFrame
		if err := json.Unmarshal(msg.Value, &frame); err != nil {
			logger.Warn("解析音频帧失败", zap.Error(err))
			continue
		}

		session := p.getOrCreateSession(frame.RoomID, frame.UserID)
		session.feedAudio(frame.Data)
	}
}

func (p *InboundPipeline) getOrCreateSession(roomID, userID string) *roomSession {
	val, _ := p.roomSessions.LoadOrStore(roomID, &roomSession{
		roomID:   roomID,
		userID:   userID,
		audioCh:  make(chan []byte, 200),
		pipeline: p,
	})

	rs := val.(*roomSession)
	rs.startOnce.Do(func() {
		rs.timer = time.AfterFunc(60*time.Second, func() {
			logger.Info("房间空闲超时，关闭音频通道", zap.String("room", rs.roomID))
			close(rs.audioCh)
		})
		go rs.runSTTLoop()
		logger.Info("创建新房间 STT 会话", zap.String("room", roomID), zap.String("user", userID))
	})

	if rs.userID != userID {
		rs.userID = userID
	}
	return rs
}

func (rs *roomSession) feedAudio(data []byte) {
	rs.mu.Lock()
	if rs.timer != nil {
		rs.timer.Reset(60 * time.Second)
	}
	rs.mu.Unlock()

	select {
	case rs.audioCh <- data:
	default:
		logger.Warn("音频通道已满，丢弃一帧", zap.String("room", rs.roomID))
	}
}

func (rs *roomSession) runSTTLoop() {
	defer func() {
		rs.pipeline.roomSessions.Delete(rs.roomID)
		logger.Info("房间 STT 会话结束", zap.String("room", rs.roomID))
	}()

	ctx := context.Background()
	textStream, err := rs.pipeline.svc.STT.TranscribeStream(ctx, rs.audioCh)
	if err != nil {
		logger.Error("STT TranscribeStream 失败", zap.String("room", rs.roomID), zap.Error(err))
		return
	}

	for sentence := range textStream {
		logger.Info("识别出句子", zap.String("room", rs.roomID), zap.String("text", truncate(sentence, 80)))

		if err := rs.pipeline.svc.TextInProd.SendText(ctx, mq.TextInMessage{
			RoomID: rs.roomID, UserID: rs.userID, Text: sentence, Timestamp: time.Now().UnixMilli(),
		}); err != nil {
			logger.Error("写入 text.in 失败", zap.String("room", rs.roomID), zap.Error(err))
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
