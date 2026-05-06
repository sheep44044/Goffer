package service

import (
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"context"
)

type CheckStatusService struct {
	svc *svc.ServiceContext
}

func NewCheckStatusService(svc *svc.ServiceContext) *CheckStatusService {
	return &CheckStatusService{
		svc: svc,
	}
}

func (s *CheckStatusService) CheckResumeStatus(ctx context.Context, req *user.CheckResumeStatusReq) (int, error) {
	status, err := s.svc.DB.GetResumeStatus(ctx, req.ResumeId, req.UserId)
	if err != nil {
		return 0, err
	}

	return status, nil
}
