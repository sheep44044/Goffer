package pipeline

import (
	"Goffer/app/rpc/earmouth/mq"
	"Goffer/app/rpc/earmouth/svc"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type TextOutMessage struct {
	RoomID    string `json:"room_id"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

type OutboundPipeline struct {
	svc *svc.ServiceContext
}

func NewOutboundPipeline(svc *svc.ServiceContext) *OutboundPipeline {
	return &OutboundPipeline{svc: svc}
}

func (p *OutboundPipeline) Start() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        p.svc.Config.Kafka.Brokers,
		GroupID:        p.svc.Config.Kafka.ConsumerGroup + "-outbound",
		Topic:          p.svc.Config.Kafka.TextOutTopic,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second,
		MinBytes:       1,
		MaxBytes:       10e6,
	})
	defer reader.Close()

	ctx := context.Background()
	logger.Info("开始消费 text.out topic", zap.String("topic", p.svc.Config.Kafka.TextOutTopic))

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			logger.Warn("拉取 Kafka 消息失败", zap.Error(err))
			continue
		}

		var textMsg TextOutMessage
		if err := json.Unmarshal(msg.Value, &textMsg); err != nil {
			logger.Warn("解析文本消息失败", zap.Error(err))
			continue
		}

		if textMsg.Text == "" || textMsg.RoomID == "" {
			continue
		}

		// 拦截 1: 消息时间戳早于打断时间 → 丢弃
		if p.svc.CancelTracker.IsStale(textMsg.RoomID, textMsg.Timestamp) {
			logger.Info("丢弃废弃 TTS 任务(已被打断)", zap.String("room", textMsg.RoomID), zap.Int64("msg_ts", textMsg.Timestamp))
			continue
		}

		logger.Info("收到 TTS 合成请求", zap.String("room", textMsg.RoomID), zap.String("text", truncateStr(textMsg.Text, 80)))

		// 拦截 2: 可取消的 TTS context
		ttsCtx, ttsCancel := context.WithCancel(ctx)
		defer ttsCancel()
		go p.watchCancelForTTS(textMsg.RoomID, textMsg.Timestamp, ttsCancel)

		audioStream, err := p.svc.TTS.SynthesizeStream(ttsCtx, textMsg.Text)
		if err != nil {
			logger.Error("TTS 合成失败", zap.String("room", textMsg.RoomID), zap.Error(err))
			continue
		}

		for audioData := range audioStream {
			if err := p.svc.AudioOutProd.SendAudio(ctx, mq.AudioOutChunk{
				RoomID: textMsg.RoomID, Timestamp: time.Now().UnixMilli(),
				Codec: "pcm", Data: audioData,
			}); err != nil {
				logger.Error("写入 audio.out 失败", zap.String("room", textMsg.RoomID), zap.Error(err))
			}
		}
	}
}

func (p *OutboundPipeline) watchCancelForTTS(roomID string, msgTS int64, cancel context.CancelFunc) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		if p.svc.CancelTracker.IsStale(roomID, msgTS) {
			logger.Info("TTS 合成被打断", zap.String("room", roomID), zap.Int64("msg_ts", msgTS))
			cancel()
			return
		}
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
