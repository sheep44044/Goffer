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

	if err := p.questionWriter.WriteMessages(ctx, kafka.Message{Value: msgBytes}); err != nil {
		return fmt.Errorf("failed to send Question msg to kafka: %w", err)
	}

	return nil
}
