package media

import (
	"Goffer/app/rpc/media/config"
	"Goffer/app/rpc/media/svc"
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

	mediaSvc := NewMediaService(svcCtx)
	mediaSvc.Start()

	logger.Info("Media 服务已启动",
		zap.Strings("stun_servers", cfg.WebRTC.STUNServers),
		zap.Int("port_min", cfg.WebRTC.PortMin),
		zap.Int("port_max", cfg.WebRTC.PortMax))

	select {}
}
