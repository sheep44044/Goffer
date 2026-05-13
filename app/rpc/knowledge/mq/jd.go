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

func (p *KafkaProducer) SendJDParseTask(task JDParseTask) error {
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
