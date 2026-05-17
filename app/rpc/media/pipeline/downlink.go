package pipeline

import (
	"Goffer/app/rpc/media/svc"
	"Goffer/app/rpc/media/webrtc"
	"Goffer/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type AudioOutChunk struct {
	RoomID    string `json:"room_id"`
	Timestamp int64  `json:"timestamp"`
	Codec     string `json:"codec"`
	Data      []byte `json:"data"`
}

type DownlinkPipeline struct {
	svc     *svc.ServiceContext
	roomMgr *webrtc.RoomManager
}

func NewDownlinkPipeline(svc *svc.ServiceContext, roomMgr *webrtc.RoomManager) *DownlinkPipeline {
	return &DownlinkPipeline{svc: svc, roomMgr: roomMgr}
}

func (d *DownlinkPipeline) Start() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        d.svc.Config.Kafka.Brokers,
		GroupID:        d.svc.Config.Kafka.ConsumerGroup + "-downlink",
		Topic:          d.svc.Config.Kafka.AudioOutTopic,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second,
		MinBytes:       1,
		MaxBytes:       10e6,
	})
	defer reader.Close()

	ctx := context.Background()
	logger.Info("开始消费 audio.out topic", zap.String("topic", d.svc.Config.Kafka.AudioOutTopic))

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			logger.Warn("拉取 Kafka 消息失败", zap.Error(err))
			continue
		}

		var chunk AudioOutChunk
		if err := json.Unmarshal(msg.Value, &chunk); err != nil {
			logger.Warn("解析音频块失败", zap.Error(err))
			continue
		}

		if chunk.RoomID == "" || len(chunk.Data) == 0 {
			continue
		}

		peers := d.roomMgr.RoomPeers(chunk.RoomID)
		if len(peers) == 0 {
			continue
		}

		for _, peer := range peers {
			if err := peer.WriteAudio(chunk.Data); err != nil {
				logger.Warn("写入下行音频失败", zap.String("room", chunk.RoomID), zap.Error(err))
			}
		}
	}
}
