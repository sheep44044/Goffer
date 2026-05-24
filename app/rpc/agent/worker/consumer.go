package worker

import (
	"Goffer/app/rpc/agent/rag/store/jd"
	"Goffer/app/rpc/agent/rag/store/question"
	"Goffer/app/rpc/agent/rag/store/resume"
	"Goffer/app/rpc/agent/svc"
	"Goffer/pkg/logger"
	"context"

	"go.uber.org/zap"
)

type MQEngine struct {
	svc *svc.ServiceContext
}

func NewMQEngine(svc *svc.ServiceContext) *MQEngine {
	return &MQEngine{svc: svc}
}

func (e *MQEngine) Start(ctx context.Context) {
	logger.Info("Starting all RAG Kafka Consumers...")

	resumeWorker, err := resume.NewResumeWorker(e.svc)
	if err != nil {
		logger.Error("Resume Worker 初始化失败", zap.Error(err))
	} else {
		go resumeWorker.Start(ctx)
	}

	questionWorker, err := question.NewQuestionWorker(e.svc)
	if err != nil {
		logger.Error("Question Worker 初始化失败", zap.Error(err))
	} else {
		go questionWorker.Start(ctx)
	}

	jdWorker, err := jd.NewJDWorker(e.svc)
	if err != nil {
		logger.Error("JD Worker 初始化失败", zap.Error(err))
	} else {
		go jdWorker.Start(ctx)
	}
}
