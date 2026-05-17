package cancelmgr

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Redis Pub/Sub 打断事件频道名（与 Media 发布端、EarMouth 订阅端保持一致）
const CancelChannel = "interview:cancel_events"

// CancelEvent 打断事件结构
type CancelEvent struct {
	RoomID    string `json:"room_id"`
	Timestamp int64  `json:"timestamp"`
}

// CancelManager 管理面试房间的可取消推理上下文。
// Agent 开始推理时注册 CancelFunc，收到 Redis 打断事件时调用 CancelFunc 终止 LLM 输出。
type CancelManager struct {
	mu        sync.RWMutex
	cancelMap map[string]context.CancelFunc // roomID → cancelFunc
}

func NewCancelManager() *CancelManager {
	return &CancelManager{
		cancelMap: make(map[string]context.CancelFunc),
	}
}

// Register 注册房间的可取消上下文。同一房间的新注册会覆盖旧的（并发安全）。
func (m *CancelManager) Register(roomID string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 如果已有旧 context，先取消再覆盖（防御性处理）
	if oldCancel, ok := m.cancelMap[roomID]; ok {
		oldCancel()
	}
	m.cancelMap[roomID] = cancel
	log.Printf("[CancelMgr] 注册可取消上下文: room=%s", roomID)
}

// Cancel 取消指定房间正在进行的推理并清理记录
func (m *CancelManager) Cancel(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.cancelMap[roomID]; ok {
		cancel() // 触发 context.CancelFunc：Eino DAG 收到 ctx.Done() 后终止
		delete(m.cancelMap, roomID)
		log.Printf("[CancelMgr] 已取消推理: room=%s", roomID)
	}
}

// Remove 清理指定房间记录（推理正常结束时调用）
func (m *CancelManager) Remove(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancelMap, roomID)
}

// ===================== Redis 订阅者 =====================

// SubscribeCancelEvents 启动 Goroutine 订阅 Redis Pub/Sub 打断事件。
// 收到事件后调用 CancelManager.Cancel 终止对应房间的 LLM 推理。
// 注意：PubSub 连接不传 context.Background()，而是用独立无超时 Context，
// 防止因为没有活跃请求导致订阅断开。
func (m *CancelManager) SubscribeCancelEvents(rdb *redis.Client) {
	go func() {
		ctx := context.Background()
		pubsub := rdb.Subscribe(ctx, CancelChannel)
		defer pubsub.Close()

		// 等待订阅确认
		if _, err := pubsub.Receive(ctx); err != nil {
			log.Printf("[CancelMgr] Redis 订阅失败: %v", err)
			return
		}
		log.Printf("[CancelMgr] 已订阅 Redis Channel: %s", CancelChannel)

		ch := pubsub.Channel()
		for msg := range ch {
			var event CancelEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("[CancelMgr] 解析打断事件失败: %v payload=%s", err, msg.Payload)
				continue
			}

			if event.RoomID == "" {
				continue
			}

			log.Printf("[CancelMgr] 收到打断事件: room=%s ts=%d", event.RoomID, event.Timestamp)
			m.Cancel(event.RoomID)
		}

		log.Printf("[CancelMgr] Redis 订阅通道已关闭")
	}()
}
