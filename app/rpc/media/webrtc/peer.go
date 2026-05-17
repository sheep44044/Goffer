package webrtc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"Goffer/app/rpc/media/mq"
	"Goffer/pkg/logger"

	pion "github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"go.uber.org/zap"
)

type Peer struct {
	pc         *pion.PeerConnection
	roomID     string
	userID     string
	kafkaProd  *mq.KafkaProducer
	audioTrack *pion.TrackLocalStaticSample

	done        chan struct{}
	cleanupOnce sync.Once

	onCleanup func(roomID, userID string)
	onCancel  func(roomID, userID string)
}

func NewPeer(
	roomID, userID string,
	api *pion.API,
	kafkaProd *mq.KafkaProducer,
	stunServers []string,
	onLocalCandidate func(candidate string),
) (*Peer, error) {
	iceServers := make([]pion.ICEServer, 0, len(stunServers))
	for _, url := range stunServers {
		iceServers = append(iceServers, pion.ICEServer{URLs: []string{url}})
	}

	pc, err := api.NewPeerConnection(pion.Configuration{ICEServers: iceServers})
	if err != nil {
		return nil, fmt.Errorf("创建 PeerConnection 失败: %w", err)
	}

	audioTrack, err := pion.NewTrackLocalStaticSample(
		pion.RTPCodecCapability{MimeType: pion.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "goffer-tts",
	)
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("创建下行音频轨道失败: %w", err)
	}

	if _, err := pc.AddTrack(audioTrack); err != nil {
		pc.Close()
		return nil, fmt.Errorf("添加下行音频轨道失败: %w", err)
	}

	p := &Peer{
		pc:         pc,
		roomID:     roomID,
		userID:     userID,
		kafkaProd:  kafkaProd,
		audioTrack: audioTrack,
		done:       make(chan struct{}),
	}

	pc.OnTrack(p.onTrack)
	pc.OnICECandidate(func(c *pion.ICECandidate) {
		if c == nil {
			return
		}
		if onLocalCandidate != nil {
			onLocalCandidate(c.ToJSON().Candidate)
		}
	})
	pc.OnDataChannel(p.onDataChannel)
	pc.OnConnectionStateChange(p.onConnectionStateChange)

	return p, nil
}

func (p *Peer) SetOnCleanup(fn func(roomID, userID string)) { p.onCleanup = fn }
func (p *Peer) SetOnCancel(fn func(roomID, userID string))  { p.onCancel = fn }

func (p *Peer) SetRemoteDescription(sdp string) error {
	return p.pc.SetRemoteDescription(pion.SessionDescription{Type: pion.SDPTypeOffer, SDP: sdp})
}

func (p *Peer) CreateAnswer() (string, error) {
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return "", err
	}
	return answer.SDP, nil
}

func (p *Peer) AddICECandidate(candidate string) error {
	return p.pc.AddICECandidate(pion.ICECandidateInit{Candidate: candidate})
}

func (p *Peer) WriteAudio(data []byte) error {
	select {
	case <-p.done:
		return fmt.Errorf("peer closed: room=%s user=%s", p.roomID, p.userID)
	default:
	}
	return p.audioTrack.WriteSample(media.Sample{Data: data, Duration: 20 * time.Millisecond})
}

func (p *Peer) Close() { p.cleanup() }

// ---------- uplink ----------

func (p *Peer) onTrack(track *pion.TrackRemote, _ *pion.RTPReceiver) {
	if track.Kind() != pion.RTPCodecTypeAudio {
		return
	}
	go p.handleAudioTrack(track)
}

func (p *Peer) handleAudioTrack(track *pion.TrackRemote) {
	codec := track.Codec().MimeType
	logger.Info("音频轨道就绪", zap.String("room", p.roomID), zap.String("user", p.userID), zap.String("codec", codec))

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			logger.Info("音频轨道关闭", zap.String("room", p.roomID), zap.String("user", p.userID), zap.Error(err))
			return
		}

		frame := mq.AudioFrame{
			RoomID: p.roomID, UserID: p.userID, Timestamp: time.Now().UnixMilli(),
			Codec: codec, Data: rtpPacket.Payload,
		}

		select {
		case <-p.done:
			return
		default:
		}

		if err := p.kafkaProd.SendAudioFrame(context.Background(), frame); err != nil {
			logger.Error("Kafka 投递音频帧失败", zap.String("room", p.roomID), zap.String("user", p.userID), zap.Error(err))
		}
	}
}

// ---------- DataChannel ----------

func (p *Peer) onDataChannel(dc *pion.DataChannel) {
	logger.Info("DataChannel 已建立", zap.String("room", p.roomID), zap.String("user", p.userID), zap.String("label", dc.Label()))

	dc.OnMessage(func(msg pion.DataChannelMessage) {
		if msg.IsString {
			text := string(msg.Data)
			logger.Info("DataChannel 收到消息", zap.String("room", p.roomID), zap.String("user", p.userID), zap.String("msg", text))

			if text == `{"action":"cancel"}` {
				logger.Info("用户打断(barge-in)", zap.String("room", p.roomID), zap.String("user", p.userID))
				if p.onCancel != nil {
					p.onCancel(p.roomID, p.userID)
				}
			}
		}
	})
}

// ---------- lifecycle ----------

func (p *Peer) onConnectionStateChange(state pion.PeerConnectionState) {
	logger.Info("连接状态变更", zap.String("room", p.roomID), zap.String("user", p.userID), zap.String("state", state.String()))

	switch state {
	case pion.PeerConnectionStateFailed, pion.PeerConnectionStateClosed:
		logger.Info("连接终态，触发资源回收", zap.String("room", p.roomID), zap.String("user", p.userID))
		p.cleanup()
	case pion.PeerConnectionStateDisconnected:
		logger.Info("ICE 断连，等待重连", zap.String("room", p.roomID), zap.String("user", p.userID))
		go func() {
			timer := time.NewTimer(30 * time.Second)
			defer timer.Stop()
			select {
			case <-timer.C:
				logger.Info("断连超时，强制回收", zap.String("room", p.roomID), zap.String("user", p.userID))
				p.cleanup()
			case <-p.done:
			}
		}()
	default:
	}
}

func (p *Peer) cleanup() {
	p.cleanupOnce.Do(func() {
		logger.Info("开始清理 Peer", zap.String("room", p.roomID), zap.String("user", p.userID))
		close(p.done)
		if p.pc != nil {
			p.pc.Close()
		}
		if p.onCleanup != nil {
			p.onCleanup(p.roomID, p.userID)
		}
		logger.Info("Peer 清理完成", zap.String("room", p.roomID), zap.String("user", p.userID))
	})
}
