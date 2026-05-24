package resume

import (
	"Goffer/kitex_gen/user"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func (w *ResumeWorker) Start(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     w.svc.Config.Kafka.Brokers,
		GroupID:     "resume-parser-group",
		Topic:       w.svc.Config.Kafka.ResumeTopic,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	logger.Info("Kafka Resume Worker 启动", zap.String("topic", w.svc.Config.Kafka.ResumeTopic))

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("Resume Worker 收到关闭信号，停止拉取消息")
				return
			}
			logger.Warn("拉取消息失败", zap.Error(err))
			continue
		}

		var task ResumeParseTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			logger.Warn("解析 JSON 失败的脏数据，已丢弃", zap.Error(err))
			reader.CommitMessages(ctx, msg)
			continue
		}

		logger.Info("收到解析任务", zap.String("resume_id", task.ResumeID))

		err = w.processWithRetry(ctx, task)
		if err != nil {
			logger.Error("任务最终失败放弃",
				zap.String("resume_id", task.ResumeID),
				zap.Error(err),
			)

			_, err = w.svc.UserClient.UpdateResumeStatus(ctx, &user.UpdateResumeStatusReq{
				ResumeId: task.ResumeID,
				Status:   -1,
			})
			if err != nil {
				logger.Warn("更新简历状态为失败时出错",
					zap.String("resume_id", task.ResumeID),
					zap.Error(err),
				)
			}
		} else {
			logger.Info("任务成功完成", zap.String("resume_id", task.ResumeID))
		}

		err = reader.CommitMessages(ctx, msg)
		if err != nil {
			logger.Warn("Commit offset 失败", zap.Error(err))
		} else {
			logger.Info("任务完成并提交",
				zap.Int("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
			)
		}
	}
}

func (w *ResumeWorker) processWithRetry(ctx context.Context, task ResumeParseTask) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = w.HandleResumeParse(ctx, task)
		if err == nil {
			return nil
		}

		logger.Warn("处理失败，准备重试",
			zap.String("resume_id", task.ResumeID),
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		if isNonRetryableError(err) {
			break
		}

		waitTime := time.Second * time.Duration(i*2+1)
		select {
		case <-ctx.Done():
			return fmt.Errorf("重试过程中被外部取消: %w", ctx.Err())
		case <-time.After(waitTime):

		}
	}

	return fmt.Errorf("超过最大重试次数: %w", err)
}

func isNonRetryableError(err error) bool {
	return false
}
