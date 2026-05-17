package media

import (
	"fmt"

	"Goffer/app/rpc/media/pipeline"
	"Goffer/app/rpc/media/signaling"
	"Goffer/app/rpc/media/svc"
	"Goffer/app/rpc/media/webrtc"
	"Goffer/pkg/logger"

	"go.uber.org/zap"
)

type MediaServiceImpl struct {
	svc     *svc.ServiceContext
	roomMgr *webrtc.RoomManager
}

func NewMediaService(svc *svc.ServiceContext) *MediaServiceImpl {
	return &MediaServiceImpl{svc: svc, roomMgr: webrtc.NewRoomManager()}
}

func (s *MediaServiceImpl) Start() {
	signalHandler := signaling.NewSignalHandler(s.svc, s.roomMgr)
	addr := fmt.Sprintf("0.0.0.0:%s", s.svc.Config.Server.Port)
	signalHandler.StartSignalServer(addr)
	logger.Info("信令服务已启动", zap.String("addr", addr))

	downlink := pipeline.NewDownlinkPipeline(s.svc, s.roomMgr)
	go downlink.Start()
	logger.Info("下行音频管道已启动: audio.out -> WebRTC")
}
