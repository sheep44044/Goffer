package main

import (
	"Goffer/app/rpc/user/pack"
	"Goffer/app/rpc/user/service"
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"go.uber.org/zap"
)

type UserServiceImpl struct {
	svc *svc.ServiceContext
}

func (s *UserServiceImpl) Register(ctx context.Context, req *user.RegisterReq) (resp *user.RegisterResp, err error) {
	resp = new(user.RegisterResp)

	if len(req.Username) == 0 || len(req.Password) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 注册请求", zap.String("username", req.Username))

	if err = service.NewRegisterService(s.svc).Register(ctx, req); err != nil {
		logger.ErrorCtx(ctx, "注册失败", zap.String("username", req.Username), zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	return resp, nil
}

func (s *UserServiceImpl) Login(ctx context.Context, req *user.LoginReq) (resp *user.LoginResp, err error) {
	resp = new(user.LoginResp)

	if len(req.Username) == 0 || len(req.Password) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 登录请求", zap.String("username", req.Username))

	token, err := service.NewLoginService(s.svc).Login(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "登录失败", zap.String("username", req.Username), zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.Token = token
	return resp, nil
}

func (s *UserServiceImpl) UploadResume(ctx context.Context, req *user.UploadResumeReq) (resp *user.UploadResumeResp, err error) {
	resp = new(user.UploadResumeResp)

	if len(req.UserId) == 0 || len(req.FileContent) == 0 || len(req.FileName) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 上传简历请求",
		zap.String("user_id", req.UserId),
		zap.String("file_name", req.FileName))

	resumeID, fileURL, err := service.NewUploadResumeService(s.svc).UploadResume(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "上传简历RPC失败",
			zap.String("user_id", req.UserId),
			zap.String("file_name", req.FileName),
			zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.ResumeId = resumeID
	resp.FileUrl = fileURL
	return resp, nil
}

func (s *UserServiceImpl) CheckResumeStatus(ctx context.Context, req *user.CheckResumeStatusReq) (resp *user.CheckResumeStatusResp, err error) {
	resp = new(user.CheckResumeStatusResp)

	if len(req.UserId) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	status, err := service.NewCheckStatusService(s.svc).CheckResumeStatus(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "查询简历状态失败",
			zap.String("user_id", req.UserId),
			zap.String("resume_id", req.ResumeId),
			zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.ParseStatus = int32(status)
	return resp, nil
}

func (s *UserServiceImpl) UpdateResumeStatus(ctx context.Context, req *user.UpdateResumeStatusReq) (resp *user.UpdateResumeStatusResp, err error) {
	resp = new(user.UpdateResumeStatusResp)

	if len(req.ResumeId) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	if err = service.NewUpdateStatusService(s.svc).UpdateResumeStatus(ctx, req); err != nil {
		logger.ErrorCtx(ctx, "更新简历状态失败",
			zap.String("resume_id", req.ResumeId),
			zap.Int64("status", int64(req.Status)),
			zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	return resp, nil
}
