package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"Goffer/app/rpc/media/svc"
	"Goffer/app/rpc/media/webrtc"
	"Goffer/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const CancelChannel = "interview:cancel_events"

type CancelEvent struct {
	RoomID    string `json:"room_id"`
	Timestamp int64  `json:"timestamp"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type SignalHandler struct {
	svcCtx  *svc.ServiceContext
	roomMgr *webrtc.RoomManager
}

func NewSignalHandler(svcCtx *svc.ServiceContext, roomMgr *webrtc.RoomManager) *SignalHandler {
	return &SignalHandler{svcCtx: svcCtx, roomMgr: roomMgr}
}

func (h *SignalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 使用 HTTP request context 保持 OTel TraceID 贯通信令阶段
	reqCtx := r.Context()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorCtx(reqCtx, "WebSocket 升级失败", zap.Error(err))
		return
	}
	defer conn.Close()

	logger.InfoCtx(reqCtx, "新信令连接", zap.String("remote", r.RemoteAddr))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Info("信令连接断开", zap.String("remote", r.RemoteAddr), zap.Error(err))
			break
		}

		var msg SignalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Warn("信令消息解析失败", zap.Error(err))
			h.sendError(conn, "", "消息格式错误")
			continue
		}

		switch msg.Type {
		case SignalOffer:
			h.handleOffer(conn, &msg)
		case SignalICECandidate:
			h.handleICECandidate(conn, &msg)
		default:
			logger.Warn("未知信令消息类型", zap.String("type", string(msg.Type)))
		}
	}
}

func (h *SignalHandler) handleOffer(conn *websocket.Conn, msg *SignalMessage) {
	if msg.RoomID == "" || msg.UserID == "" {
		h.sendError(conn, "", "缺少 room_id 或 user_id")
		return
	}

	onLocalCandidate := func(candidate string) {
		h.sendJSON(conn, SignalMessage{
			Type:      SignalICECandidate,
			RoomID:    msg.RoomID,
			UserID:    msg.UserID,
			Candidate: candidate,
		})
	}

	peer, err := webrtc.NewPeer(
		msg.RoomID, msg.UserID, h.svcCtx.WebRTCAPI, h.svcCtx.KafkaProducer,
		h.svcCtx.Config.WebRTC.STUNServers, onLocalCandidate,
	)
	if err != nil {
		logger.Error("创建 PeerConnection 失败",
			zap.String("room", msg.RoomID), zap.String("user", msg.UserID), zap.Error(err))
		h.sendError(conn, msg.RoomID, "创建连接失败")
		return
	}

	if err := peer.SetRemoteDescription(msg.SDP); err != nil {
		logger.Error("SetRemoteDescription 失败",
			zap.String("room", msg.RoomID), zap.String("user", msg.UserID), zap.Error(err))
		peer.Close()
		h.sendError(conn, msg.RoomID, "SDP 协商失败")
		return
	}

	answerSDP, err := peer.CreateAnswer()
	if err != nil {
		logger.Error("CreateAnswer 失败",
			zap.String("room", msg.RoomID), zap.String("user", msg.UserID), zap.Error(err))
		peer.Close()
		h.sendError(conn, msg.RoomID, "生成 Answer 失败")
		return
	}

	redisCli := h.svcCtx.RedisClient
	peer.SetOnCancel(func(rid, uid string) {
		event := CancelEvent{RoomID: rid, Timestamp: time.Now().UnixMilli()}
		payload, _ := json.Marshal(event)
		if err := redisCli.Publish(context.Background(), CancelChannel, payload).Err(); err != nil {
			logger.Error("Redis 发布打断事件失败", zap.String("room", rid), zap.Error(err))
		} else {
			logger.Info("Redis 打断事件已广播", zap.String("room", rid), zap.Int64("ts", event.Timestamp))
		}
	})

	h.roomMgr.AddPeer(msg.RoomID, msg.UserID, peer)

	h.sendJSON(conn, SignalMessage{
		Type: SignalAnswer, RoomID: msg.RoomID, UserID: msg.UserID, SDP: answerSDP,
	})

	logger.Info("Offer 协商完成", zap.String("room", msg.RoomID), zap.String("user", msg.UserID))
}

func (h *SignalHandler) handleICECandidate(_ *websocket.Conn, msg *SignalMessage) {
	peer, ok := h.roomMgr.GetPeer(msg.RoomID, msg.UserID)
	if !ok {
		logger.Warn("ICE 未找到 Peer", zap.String("room", msg.RoomID), zap.String("user", msg.UserID))
		return
	}
	if msg.Candidate == "" {
		return
	}
	if err := peer.AddICECandidate(msg.Candidate); err != nil {
		logger.Warn("AddICECandidate 失败",
			zap.String("room", msg.RoomID), zap.String("user", msg.UserID), zap.Error(err))
	}
}

func (h *SignalHandler) sendError(conn *websocket.Conn, roomID, errMsg string) {
	h.sendJSON(conn, SignalMessage{Type: SignalError, RoomID: roomID, Error: errMsg})
}

func (h *SignalHandler) sendJSON(conn *websocket.Conn, msg SignalMessage) {
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Warn("信令发送消息失败", zap.Error(err))
	}
}

func (h *SignalHandler) StartSignalServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWebSocket)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		logger.Info("HTTP 信令服务启动", zap.String("addr", "http://"+addr+"/ws"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("信令服务异常退出", zap.Error(err))
		}
	}()
}
