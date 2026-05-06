package service

import (
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errno.ResumeNotFoundErr
		}
		return 0, fmt.Errorf("query resume status failed: %w", err)
	}

	return status, nil
}
