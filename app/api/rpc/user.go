package rpc

import (
	"Goffer/kitex_gen/user"
	"Goffer/kitex_gen/user/userservice"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"go.uber.org/zap"
)

var userClient userservice.Client

func Register(ctx context.Context, req *user.RegisterReq) error {
	resp, err := userClient.Register(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 User.Register 失败", zap.String("username", req.Username), zap.Error(err))
		return err
	}

	if resp.Resp.Code != 0 {
		return errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return nil
}

func Login(ctx context.Context, req *user.LoginReq) (*user.LoginResp, error) {
	resp, err := userClient.Login(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 User.Login 失败", zap.String("username", req.Username), zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}

func UploadResume(ctx context.Context, req *user.UploadResumeReq) (*user.UploadResumeResp, error) {
	resp, err := userClient.UploadResume(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 User.UploadResume 失败",
			zap.String("user_id", req.UserId),
			zap.String("file_name", req.FileName),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}
