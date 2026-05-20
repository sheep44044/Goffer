package mq

import (
	"Goffer/app/rpc/user/config"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

// InitProducer 初始化 Kafka 生产者
func InitProducer(cfg *config.Config) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Topic:        cfg.Kafka.Topic,
		Balancer:     &kafka.LeastBytes{}, // 类似于随机打散，平衡负载
		RequiredAcks: kafka.RequireAll,    // 等待所有副本确认，保证数据不丢失
		// Async: false, 默认就是同步发送
		MaxAttempts:  3,
		WriteTimeout: 10 * time.Second,
	}

	return &KafkaProducer{
		writer: writer,
	}
}

// ParseTask 定义投递到 Kafka 的消息体 (保持轻量)
type ParseTask struct {
	ResumeID string `json:"resume_id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
}

// SendResumeParseTask 发送简历解析任务
func (p *KafkaProducer) SendResumeParseTask(ctx context.Context, task ParseTask) error {
	msgBytes, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal ParseTask: %w", err)
	}

	msg := kafka.Message{
		Value: msgBytes,
	}

	// 发送消息，使用 context 控制超时等
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send msg to kafka: %w", err)
	}

	logger.InfoCtx(ctx, "Task sent successfully to kafka topic: resume_parse_topic",
		zap.String("resume_id", task.ResumeID),
		zap.String("file_type", task.FileType),
	)

	return nil
}

func (p *KafkaProducer) CloseProducer() {
	if p.writer != nil {
		p.writer.Close()
	}
}
