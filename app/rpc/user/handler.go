package user

import (
	"Goffer/app/rpc/user/pack"
	"Goffer/app/rpc/user/service"
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"context"
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

	err = service.NewRegisterService(s.svc).Register(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
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

	token, err := service.NewLoginService(s.svc).Login(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
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

	resumeID, fileURL, err := service.NewUploadResumeService(s.svc).UploadResume(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
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
		resp.Resp = pack.BuildBaseResp(err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.ParseStatus = int32(status)
	return resp, nil
}
