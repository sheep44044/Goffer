package resume

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/user"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type ResumeWorker struct {
	svc *svc.ServiceContext
}

func NewResumeWorker(svc *svc.ServiceContext) *ResumeWorker {
	return &ResumeWorker{
		svc: svc,
	}
}

func (w *ResumeWorker) Start() {
	// 从 svcCtx 的 Config 中读取 Kafka 配置
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     w.svc.Config.Kafka.Brokers,
		GroupID:     "resume-parser-group", // 或者从配置读取
		Topic:       w.svc.Config.Kafka.ResumeTopic,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	ctx := context.Background() // 后台任务的根 Context
	fmt.Println("Kafka Resume Worker is running in background...")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("Error fetching message: %v", err)
			continue
		}

		var task ResumeParseTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			log.Printf("解析 JSON 失败的脏数据: %v", err)
			reader.CommitMessages(ctx, msg) // 脏数据直接提交丢弃
			continue
		}

		fmt.Printf("收到解析任务... ResumeID: %s\n", task.ResumeID)

		// 调用处理函数，把 svc 和 ctx 传进去
		err = w.HandleResumeParse(ctx, task)
		if err != nil {
			// 经过多次重试依然失败，进入死信处理流程
			log.Printf("【严重异常】任务最终失败放弃 (ResumeID: %s): %v", task.ResumeID, err)

			// 1. 更新数据库状态为“解析失败” (非常重要，否则前端一直显示解析中)
			_, err = w.svc.UserClient.UpdateResumeStatus(ctx, &user.UpdateResumeStatusReq{
				ResumeId: task.ResumeID,
				Status:   -1,
			}) // 假设 -1 表示失败
			// 2. (可选) 将这条 task 序列化后发往另一个专门存错误的 Kafka Topic (死信队列 DLQ)，或者记录到 MySQL 的 error_log 表
		} else {
			fmt.Printf("任务成功完成 (ResumeID: %s)\n", task.ResumeID)
		}

		// 成功处理，提交 offset
		err = reader.CommitMessages(ctx, msg)
		if err != nil {
			log.Printf("Commit offset 失败: %v", err)
		} else {
			fmt.Printf("任务完成并提交 (Partition: %d, Offset: %d)\n", msg.Partition, msg.Offset)
		}
	}
}

func (w *ResumeWorker) processWithRetry(ctx context.Context, task ResumeParseTask) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = w.HandleResumeParse(ctx, task)
		if err == nil {
			return nil // 成功直接返回
		}

		log.Printf("第 %d 次处理失败 (ResumeID: %s): %v", i+1, task.ResumeID, err)

		// 针对一些不可恢复的错误（比如文件不存在，或者 ID 错误），直接退出，不重试
		if isNonRetryableError(err) {
			break
		}

		// 等待一段时间再重试，防止由于网络瞬间抖动导致的密集报错
		time.Sleep(time.Second * time.Duration(i*2+1))
	}

	return fmt.Errorf("超过最大重试次数: %w", err)
}

// 辅助函数：判断是否是无需重试的致命错误
func isNonRetryableError(err error) bool {
	// 比如判断错误类型：如果是 Minio 找不到文件 (404)，那么重试 100 次也没用，直接跳过
	// 如果是网络超时，那就值得重试
	return false
}
