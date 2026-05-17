package mq

import (
	"Goffer/app/rpc/media/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

// InitProducer 初始化 Kafka 生产者，向指定 topic 投递音频数据
func InitProducer(cfg *config.Config, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:  kafka.TCP(cfg.Kafka.Brokers...),
		Topic: topic,
		// 优化 1: 使用 Hash 负载均衡，配合 Message 的 Key 使用
		Balancer: &kafka.Hash{},
		// 优化 2: 实时音频流对低延迟要求极高，允许极少量丢包，不需要等待所有副本确认
		RequiredAcks: kafka.RequireNone,
		MaxAttempts:  3,
		WriteTimeout: 2 * time.Second, // 优化 3: 音频流没必要等 10s，如果超时应当快速失败丢弃该帧
	}

	return &KafkaProducer{
		writer: writer,
	}
}

// AudioFrame 投递到 Kafka 的音频帧消息体
type AudioFrame struct {
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
	Codec     string `json:"codec"` // "opus", "pcm"
	Data      []byte `json:"data"`  // 注意: JSON 序列化时这里会被自动 Base64 编码
}

// SendAudioFrame 发送单帧音频数据到 Kafka
func (p *KafkaProducer) SendAudioFrame(ctx context.Context, frame AudioFrame) error {
	msgBytes, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("failed to marshal audio frame: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(frame.RoomID), // 优化 1 核心: 必须用 RoomID 作为 Key，保证同一场面试音频包严格有序
		Value: msgBytes,
	}

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send audio frame to kafka: %w", err)
	}

	return nil
}

func (p *KafkaProducer) CloseProducer() {
	if p.writer != nil {
		p.writer.Close()
	}
}
