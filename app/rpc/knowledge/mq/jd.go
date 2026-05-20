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
	msgBytes, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal ParseTask: %w", err)
	}

	if err := p.jdWriter.WriteMessages(ctx, kafka.Message{Value: msgBytes}); err != nil {
		return fmt.Errorf("failed to send JD msg to kafka: %w", err)
	}

	return nil
}
