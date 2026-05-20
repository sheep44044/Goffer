package main

import (
	"Goffer/app/rpc/earmouth/config"
	"Goffer/app/rpc/earmouth/svc"
	"Goffer/pkg/logger"
	"Goffer/pkg/telemetry"
	"context"
	"fmt"
	"log"
	"time"

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

	shutdown, err := telemetry.InitOTel(cfg.Service.Name, "localhost:4317")
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry telemetry: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if shutdownErr := shutdown(ctx); shutdownErr != nil {
			logger.WarnCtx(ctx, "Warning: OpenTelemetry shutdown failed", zap.Error(shutdownErr))
		} else {
			logger.WarnCtx(ctx, "OpenTelemetry telemetry tracing flush and shutdown cleanly.")
		}
	}()

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
