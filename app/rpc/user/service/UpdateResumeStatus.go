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

type UpdateStatusService struct {
	svc *svc.ServiceContext
}

func NewUpdateStatusService(svc *svc.ServiceContext) *UpdateStatusService {
	return &UpdateStatusService{
		svc: svc,
	}
}

func (s *UpdateStatusService) UpdateResumeStatus(ctx context.Context, req *user.UpdateResumeStatusReq) error {
	err := s.svc.DB.UpdateResumeStatus(ctx, req.ResumeId, int(req.Status))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ResumeNotFoundErr
		}
		return fmt.Errorf("update resume status failed: %w", err)
	}

	return nil
}
