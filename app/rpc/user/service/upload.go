package service

import (
	"Goffer/app/rpc/user/dal/db"
	"Goffer/app/rpc/user/mq"
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"Goffer/pkg/snowflake"
	"context"
	"fmt"
)

type UploadResumeService struct {
	svc *svc.ServiceContext
}

func NewUploadResumeService(svc *svc.ServiceContext) *UploadResumeService {
	return &UploadResumeService{svc: svc}
}

func (s *UploadResumeService) UploadResume(ctx context.Context, req *user.UploadResumeReq) (string, string, error) {
	safeFileName, fileURL, err := s.svc.Minio.UploadFile(
		ctx,
		req.FileName,
		req.FileContent,
		req.ContentType,
	)
	if err != nil {
		return "", "", fmt.Errorf("minio upload failed: %w", err)
	}

	resumeID := snowflake.GenString()
	err = s.svc.DB.CreateResume(ctx, []*db.Resume{{
		ID:       resumeID,
		UserID:   req.UserId,
		FileURL:  fileURL,
		FileName: safeFileName,
	}})
	if err != nil {
		return "", "", fmt.Errorf("db create resume failed: %w", err)
	}

	err = s.svc.Kafka.SendResumeParseTask(ctx, mq.ParseTask{
		ResumeID: resumeID,
		FileURL:  fileURL,
		FileType: req.ContentType,
	})
	if err != nil {
		return "", "", fmt.Errorf("kafka publish failed: %w", errno.ServiceErr.WithMessage("解析任务投递失败，请稍后重试"))
	}

	return resumeID, fileURL, nil
}
