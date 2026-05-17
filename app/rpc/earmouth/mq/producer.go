package mq

import (
	"Goffer/app/rpc/earmouth/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer 通用的 Kafka 生产者封装
type Producer struct {
	writer *kafka.Writer
}

// NewProducer 创建指定 topic 的生产者
func NewProducer(cfg *config.Config, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:  kafka.TCP(cfg.Kafka.Brokers...),
		Topic: topic,
		// 使用 RoomID 作为 Key 的 Hash 负载均衡，保证同房间消息有序
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireNone, // 实时流场景优先低延迟
		MaxAttempts:  3,
		WriteTimeout: 2 * time.Second,
	}

	return &Producer{writer: writer}
}

// TextInMessage 投递到 text.in 的识别结果
type TextInMessage struct {
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

// SendText 发送 STT 识别出的句子到 text.in
func (p *Producer) SendText(ctx context.Context, msg TextInMessage) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal text message: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.RoomID),
		Value: msgBytes,
	})
}

// AudioOutChunk 投递到 audio.out 的合成音频块
type AudioOutChunk struct {
	RoomID    string `json:"room_id"`
	Timestamp int64  `json:"timestamp"`
	Codec     string `json:"codec"`
	Data      []byte `json:"data"`
}

// SendAudio 发送 TTS 合成音频到 audio.out
func (p *Producer) SendAudio(ctx context.Context, chunk AudioOutChunk) error {
	msgBytes, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("marshal audio chunk: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(chunk.RoomID),
		Value: msgBytes,
	})
}

// CloseProducer 关闭底层 Writer
func (p *Producer) CloseProducer() {
	if p.writer != nil {
		p.writer.Close()
	}
}
