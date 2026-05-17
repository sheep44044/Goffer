package svc

import (
	"Goffer/app/rpc/media/config"
	"Goffer/app/rpc/media/mq"
	"fmt"

	"github.com/pion/webrtc/v4"
	"github.com/redis/go-redis/v9"
)

// ServiceContext 统一注入 media 服务所需的全部依赖
type ServiceContext struct {
	Config        *config.Config
	KafkaProducer *mq.KafkaProducer
	RedisClient   *redis.Client // 用于发布打断事件到 Redis Pub/Sub
	WebRTCAPI     *webrtc.API
	SettingEngine *webrtc.SettingEngine
}

func NewServiceContext(cfg *config.Config) *ServiceContext {
	// 1. 初始化 Kafka Producer，对接音频上行链路
	audioProducer := mq.InitProducer(cfg, cfg.Kafka.AudioInTopic)

	// 2. 初始化 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 3. 配置 Pion SettingEngine (UDP 端口范围)
	se := &webrtc.SettingEngine{}
	if cfg.WebRTC.PortMin > 0 && cfg.WebRTC.PortMax > 0 {
		se.SetEphemeralUDPPortRange(
			uint16(cfg.WebRTC.PortMin),
			uint16(cfg.WebRTC.PortMax),
		)
	}

	// 4. 创建 WebRTC API 实例
	api := webrtc.NewAPI(webrtc.WithSettingEngine(*se))

	return &ServiceContext{
		Config:        cfg,
		KafkaProducer: audioProducer,
		RedisClient:   rdb,
		WebRTCAPI:     api,
		SettingEngine: se,
	}
}
