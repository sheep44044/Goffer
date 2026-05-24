package cancelmgr

import (
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const CancelChannel = "interview:cancel_events"

type CancelEvent struct {
	RoomID    string `json:"room_id"`
	Timestamp int64  `json:"timestamp"`
}

type CancelManager struct {
	mu        sync.RWMutex
	cancelMap map[string]context.CancelFunc
}

func NewCancelManager() *CancelManager {
	return &CancelManager{
		cancelMap: make(map[string]context.CancelFunc),
	}
}

func (m *CancelManager) Register(roomID string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if oldCancel, ok := m.cancelMap[roomID]; ok {
		oldCancel()
	}
	m.cancelMap[roomID] = cancel
	logger.Debug("注册可取消上下文", zap.String("room", roomID))
}

func (m *CancelManager) Cancel(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.cancelMap[roomID]; ok {
		cancel()
		delete(m.cancelMap, roomID)
		logger.Info("已取消推理", zap.String("room", roomID))
	}
}

func (m *CancelManager) Remove(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancelMap, roomID)
}

func (m *CancelManager) SubscribeCancelEvents(rdb *redis.Client) {
	go func() {
		ctx := context.Background()
		pubsub := rdb.Subscribe(ctx, CancelChannel)
		defer pubsub.Close()

		if _, err := pubsub.Receive(ctx); err != nil {
			logger.Error("Redis 打断订阅失败", zap.Error(err))
			return
		}
		logger.Info("已订阅 Redis 打断频道", zap.String("channel", CancelChannel))

		ch := pubsub.Channel()
		for msg := range ch {
			var event CancelEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				logger.Warn("解析打断事件失败", zap.String("payload", msg.Payload), zap.Error(err))
				continue
			}

			if event.RoomID == "" {
				continue
			}

			logger.Info("收到打断事件", zap.String("room", event.RoomID), zap.Int64("ts", event.Timestamp))
			m.Cancel(event.RoomID)
		}

		logger.Info("Redis 打断订阅通道已关闭")
	}()
}
