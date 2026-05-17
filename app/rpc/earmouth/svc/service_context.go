package svc

import (
	"Goffer/app/rpc/earmouth/config"
	"Goffer/app/rpc/earmouth/mq"
	"Goffer/app/rpc/earmouth/stt"
	"Goffer/app/rpc/earmouth/tts"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

// ServiceContext 统一注入 EarMouth 服务所需的全部依赖
type ServiceContext struct {
	Config        *config.Config
	TextInProd    *mq.Producer // 写入 interview.text.in（识别结果）
	AudioOutProd  *mq.Producer // 写入 interview.audio.out（合成音频）
	STT           stt.STTProvider
	TTS           tts.TTSProvider
	RedisClient   *redis.Client
	CancelTracker *CancelTracker // 房间级打断时间戳追踪
}

// CancelTracker 线程安全地记录每个 RoomID 的最后一次打断时间戳。
// Outbound 管道在消费 text.out 时据此丢弃废弃消息，TTS 据此中断合成。
type CancelTracker struct {
	mu     sync.RWMutex
	roomTS map[string]int64 // roomID → 最后一次打断的 UnixMilli 时间戳
}

func NewCancelTracker() *CancelTracker {
	return &CancelTracker{roomTS: make(map[string]int64)}
}

// Record 记录房间的打断时间戳
func (ct *CancelTracker) Record(roomID string, ts int64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	// 只保留最大时间戳（防乱序到达）
	if old, ok := ct.roomTS[roomID]; !ok || ts > old {
		ct.roomTS[roomID] = ts
	}
}

// IsStale 判断给定的消息时间戳是否早于最后一次打断时间（即该消息已废弃）
func (ct *CancelTracker) IsStale(roomID string, msgTS int64) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	cancelTS, ok := ct.roomTS[roomID]
	return ok && msgTS <= cancelTS
}

// Cleanup 清理指定房间的记录（面试结束时调用）
func (ct *CancelTracker) Cleanup(roomID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	delete(ct.roomTS, roomID)
}

func NewServiceContext(cfg *config.Config) *ServiceContext {
	// 1. Kafka 生产者：将 STT 识别文字写入 text.in
	textInProducer := mq.NewProducer(cfg, cfg.Kafka.TextInTopic)

	// 2. Kafka 生产者：将 TTS 合成音频写入 audio.out
	audioOutProducer := mq.NewProducer(cfg, cfg.Kafka.AudioOutTopic)

	// 3. STT/TTS 服务商
	sttProvider, err := stt.NewSTTProvider(cfg.STT.Provider)
	if err != nil {
		panic(err)
	}

	ttsProvider, err := tts.NewTTSProvider(cfg.TTS.Provider)
	if err != nil {
		panic(err)
	}

	// 4. Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return &ServiceContext{
		Config:        cfg,
		TextInProd:    textInProducer,
		AudioOutProd:  audioOutProducer,
		STT:           sttProvider,
		TTS:           ttsProvider,
		RedisClient:   redisClient,
		CancelTracker: NewCancelTracker(),
	}
}
