package mq

import (
	"Goffer/app/rpc/knowledge/config"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	jdWriter       *kafka.Writer
	questionWriter *kafka.Writer
}

func InitProducer(cfg *config.Config) *KafkaProducer {
	jdWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Topic:        cfg.Kafka.JDTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  3,
		WriteTimeout: 10 * time.Second,
	}

	questionWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Topic:        cfg.Kafka.QuestionTopic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  3,
		WriteTimeout: 10 * time.Second,
	}

	return &KafkaProducer{jdWriter: jdWriter, questionWriter: questionWriter}
}

func (p *KafkaProducer) JDBroker() *kafka.Writer       { return p.jdWriter }
func (p *KafkaProducer) QuestionBroker() *kafka.Writer { return p.questionWriter }

func (p *KafkaProducer) CloseProducer() {
	if p.jdWriter != nil {
		p.jdWriter.Close()
	}
	if p.questionWriter != nil {
		p.questionWriter.Close()
	}
}
