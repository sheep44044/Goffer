package question

import (
	"Goffer/app/rpc/agent/svc"
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type QuestionWorker struct {
	svc *svc.ServiceContext
}

func NewQuestionWorker(svc *svc.ServiceContext) *QuestionWorker {
	return &QuestionWorker{
		svc: svc,
	}
}

func (w *QuestionWorker) Start() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     w.svc.Config.Kafka.Brokers,
		GroupID:     "rag-question-group",
		Topic:       w.svc.Config.Kafka.QuestionTopic,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	ctx := context.Background()
	log.Printf("[Question Worker] 启动成功，正在独立监听 Topic: %s\n", w.svc.Config.Kafka.QuestionTopic)

	for {
		// 3. 拉取消息
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("[Question Worker] 拉取消息失败: %v", err)
			continue
		}

		// 4. 将拿到的字节流 (msg.Value) 直接交给对应的 Handler 处理
		// 这里调用的是咱们刚才修复好的那个方法
		err = w.IngestQuestion(ctx, msg.Value)
		if err != nil {
			// 处理失败：通常这里建议只记录错误日志，或者将 msg.Value 扔进一个 "死信队列 (DLQ Topic)"
			// 不要在这里一直死循环重试，否则遇到“毒药消息（脏数据）”会把整个消费组卡死
			log.Printf("[严重] 处理题库消息失败 (Topic: %s, Offset: %d): %v", msg.Topic, msg.Offset, err)
		} else {
			log.Printf("[Question Worker] 成功处理题库消息 (Offset: %d)", msg.Offset)
		}

		// 5. 提交 Offset (确认消费)
		// 无论上面的业务逻辑成功还是失败，都要往前走，避免发生阻塞。
		err = reader.CommitMessages(ctx, msg)
		if err != nil {
			log.Printf("[Question Worker] 提交 Offset 失败: %v", err)
		}
	}
}
