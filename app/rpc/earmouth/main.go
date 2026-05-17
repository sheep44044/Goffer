package earmouth

import (
	"Goffer/app/rpc/earmouth/config"
	"Goffer/app/rpc/earmouth/svc"
	"Goffer/pkg/logger"
	"Goffer/pkg/telemetry"
	"context"
	"fmt"

	"go.uber.org/zap"
)

func Init(cfg *config.Config) *svc.ServiceContext {
	return svc.NewServiceContext(cfg)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("配置加载失败: %w", err))
	}

	shutdown, _ := telemetry.InitOTel(cfg.Service.Name, "localhost:4317")
	defer shutdown(context.Background())

	svcCtx := Init(cfg)

	earMouthSvc := NewEarMouthService(svcCtx)
	earMouthSvc.Start()

	logger.Info("EarMouth 服务已启动",
		zap.String("stt_provider", cfg.STT.Provider),
		zap.String("tts_provider", cfg.TTS.Provider))
	logger.Info("Kafka Topics",
		zap.String("audio_in", cfg.Kafka.AudioInTopic),
		zap.String("text_in", cfg.Kafka.TextInTopic),
		zap.String("text_out", cfg.Kafka.TextOutTopic),
		zap.String("audio_out", cfg.Kafka.AudioOutTopic))

	select {}
}
