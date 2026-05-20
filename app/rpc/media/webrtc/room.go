package webrtc

import (
	"sync"

	"Goffer/pkg/logger"

	"go.uber.org/zap"
)

type RoomManager struct {
	mu    sync.RWMutex
	peers map[string]*Peer
}

func NewRoomManager() *RoomManager {
	return &RoomManager{peers: make(map[string]*Peer)}
}

func peerKey(roomID, userID string) string { return roomID + ":" + userID }

func (rm *RoomManager) AddPeer(roomID, userID string, peer *Peer) {
	rm.mu.Lock()
	key := peerKey(roomID, userID)
	if old, ok := rm.peers[key]; ok {
		rm.mu.Unlock()
		old.Close()
		rm.mu.Lock()
		key = peerKey(roomID, userID)
	}
	rm.peers[key] = peer
	rm.mu.Unlock()

	peer.SetOnCleanup(func(rid, uid string) {
		rm.RemovePeerSilent(rid, uid)
	})

	logger.Info("注册 Peer", zap.String("room", roomID), zap.String("user", userID))
}

func (rm *RoomManager) GetPeer(roomID, userID string) (*Peer, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	peer, ok := rm.peers[peerKey(roomID, userID)]
	return peer, ok
}

func (rm *RoomManager) RemovePeer(roomID, userID string) {
	rm.mu.Lock()
	key := peerKey(roomID, userID)
	peer, ok := rm.peers[key]
	delete(rm.peers, key)
	rm.mu.Unlock()
	if ok {
		logger.Info("主动移除 Peer", zap.String("room", roomID), zap.String("user", userID))
		peer.Close()
	}
}

func (rm *RoomManager) RemovePeerSilent(roomID, userID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.peers, peerKey(roomID, userID))
}

func (rm *RoomManager) CloseRoom(roomID string) {
	rm.mu.Lock()
	prefix := roomID + ":"
	var peersToClose []*Peer
	for key, peer := range rm.peers {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			peersToClose = append(peersToClose, peer)
			delete(rm.peers, key)
		}
	}
	rm.mu.Unlock()

	for _, peer := range peersToClose {
		peer.Close()
	}
	logger.Info("关闭房间", zap.String("room", roomID), zap.Int("peers", len(peersToClose)))
}

func (rm *RoomManager) RoomPeers(roomID string) []*Peer {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	var result []*Peer
	prefix := roomID + ":"
	for key, peer := range rm.peers {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			result = append(result, peer)
		}
	}
	return result
}
