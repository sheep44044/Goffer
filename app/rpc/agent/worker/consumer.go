package worker

import (
	"Goffer/app/rpc/agent/rag/store/jd"
	"Goffer/app/rpc/agent/rag/store/question"
	"Goffer/app/rpc/agent/rag/store/resume"
	"Goffer/app/rpc/agent/svc"
	"fmt"
	"sync"
)

type MQEngine struct {
	svc *svc.ServiceContext
}

func NewMQEngine(svc *svc.ServiceContext) *MQEngine {
	return &MQEngine{svc: svc}
}

func (e *MQEngine) Start() {
	var wg sync.WaitGroup
	fmt.Println("Starting all RAG Kafka Consumers...")

	wg.Add(1)
	go func() {
		defer wg.Done()
		resumeWorker := resume.NewResumeWorker(e.svc)
		resumeWorker.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		questionWorker := question.NewQuestionWorker(e.svc)
		questionWorker.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		jdWorker := jd.NewJDWorker(e.svc)
		jdWorker.Start()
	}()

	wg.Wait()
}
