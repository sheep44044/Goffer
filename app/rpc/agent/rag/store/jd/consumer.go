package jd

import (
	"Goffer/app/rpc/agent/svc"
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type JDWorker struct {
	svc *svc.ServiceContext
}

func NewJDWorker(svc *svc.ServiceContext) *JDWorker {
	return &JDWorker{
		svc: svc,
	}
}

func (w *JDWorker) Start() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     w.svc.Config.Kafka.Brokers,
		GroupID:     "rag-jd-group",
		Topic:       w.svc.Config.Kafka.JDTopic,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	ctx := context.Background()
	log.Printf("[JD Worker] 启动成功，正在独立监听 Topic: %s\n", w.svc.Config.Kafka.JDTopic)

	for {
		// 3. 拉取消息
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("[JD Worker] 拉取消息失败: %v", err)
			continue
		}

		// 4. 交给对应的 Handler 处理
		err = w.IngestJD(ctx, msg.Value)
		if err != nil {
			log.Printf("[严重] 处理 JD 消息失败 (Topic: %s, Offset: %d): %v", msg.Topic, msg.Offset, err)
		} else {
			log.Printf("[JD Worker] 成功处理 JD 消息 (Offset: %d)", msg.Offset)
		}

		// 5. 提交 Offset (确认消费)
		err = reader.CommitMessages(ctx, msg)
		if err != nil {
			log.Printf("[JD Worker] 提交 Offset 失败: %v", err)
		}
	}
}
