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

func (p *KafkaProducer) SendQuestionParseTask(task QuestionParseTask) error {
	msgBytes, _ := json.Marshal(task)

	msg := kafka.Message{
		Value: msgBytes,
	}

	err := p.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to send msg to kafka: %w", err)
	}

	// 注意：kafka-go 的 WriteMessages 不会直接返回具体的 partition 和 offset
	fmt.Println("Task sent successfully to kafka topic: resume_parse_topic")
	return nil
}
