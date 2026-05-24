package question

import (
	"Goffer/pkg/logger"
	"context"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func (w *QuestionWorker) Start(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     w.svc.Config.Kafka.Brokers,
		GroupID:     "rag-question-group",
		Topic:       w.svc.Config.Kafka.QuestionTopic,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	logger.Info("Question Worker 启动", zap.String("topic", w.svc.Config.Kafka.QuestionTopic))

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("Question Worker 收到关闭信号，停止拉取消息")
				return
			}
			logger.Warn("Question Worker 拉取消息失败", zap.Error(err))
			continue
		}

		err = w.IngestQuestion(ctx, msg.Value)
		if err != nil {
			logger.Error("处理题库消息失败",
				zap.String("topic", msg.Topic),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
		} else {
			logger.Info("Question Worker 成功处理消息", zap.Int64("offset", msg.Offset))
		}

		err = reader.CommitMessages(ctx, msg)
		if err != nil {
			logger.Warn("Question Worker 提交 Offset 失败", zap.Error(err))
		}
	}
}
