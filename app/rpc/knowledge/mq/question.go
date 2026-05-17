package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type QuestionParseTask struct {
	QuestionID      string   `json:"question_id"`
	QuestionContent string   `json:"question_content"`
	StandardAnswer  string   `json:"standard_answer"`
	Tags            []string `json:"tags"`
	Difficulty      string   `json:"difficulty"`
}

func (p *KafkaProducer) SendQuestionParseTask(ctx context.Context, task QuestionParseTask) error {
	msgBytes, _ := json.Marshal(task)

	msg := kafka.Message{
		Value: msgBytes,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to send msg to kafka: %w", err)
	}

	return nil
}
