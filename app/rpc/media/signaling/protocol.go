package signaling

// SignalType 定义信令消息类型
type SignalType string

const (
	SignalOffer        SignalType = "offer"
	SignalAnswer       SignalType = "answer"
	SignalICECandidate SignalType = "ice-candidate"
	SignalError        SignalType = "error"
)

// SignalMessage WebSocket 信令交换时使用的 JSON 消息体
type SignalMessage struct {
	Type      SignalType `json:"type"`
	RoomID    string     `json:"room_id"`
	UserID    string     `json:"user_id"`
	SDP       string     `json:"sdp,omitempty"`
	Candidate string     `json:"candidate,omitempty"`
	Error     string     `json:"error,omitempty"`
}
