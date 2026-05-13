package mq

import (
	"Goffer/app/rpc/knowledge/config"
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

// 建议在服务关闭时调用此方法清理资源
func (p *KafkaProducer) CloseProducer() {
	if p.writer != nil {
		p.writer.Close()
	}
}
