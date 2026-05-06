package mq

import (
	"Goffer/app/rpc/user/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
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
func (p *KafkaProducer) SendResumeParseTask(task ParseTask) error {
	msgBytes, _ := json.Marshal(task)

	msg := kafka.Message{
		Value: msgBytes,
	}

	// 发送消息，使用 context 控制超时等
	err := p.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to send msg to kafka: %w", err)
	}

	// 注意：kafka-go 的 WriteMessages 不会直接返回具体的 partition 和 offset
	fmt.Println("Task sent successfully to kafka topic: resume_parse_topic")
	return nil
}

// 建议在服务关闭时调用此方法清理资源
func (p *KafkaProducer) CloseProducer() {
	if p.writer != nil {
		p.writer.Close()
	}
}
