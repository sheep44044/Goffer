package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type JDParseTask struct {
	JDID             string   `json:"jd_id"`
	Company          string   `json:"company"`
	Title            string   `json:"title"`
	Responsibilities string   `json:"responsibilities"`
	Requirements     string   `json:"requirements"`
	Tags             []string `json:"tags"`
}

func (p *KafkaProducer) SendJDParseTask(ctx context.Context, task JDParseTask) error {
	msgBytes, _ := json.Marshal(task)

	msg := kafka.Message{
		Value: msgBytes,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to send msg to kafka: %w", err)
	}

	return nil
}
