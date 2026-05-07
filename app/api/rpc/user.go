package rpc

import (
	"Goffer/kitex_gen/user"
	"Goffer/kitex_gen/user/userservice"
	"Goffer/pkg/errno"
	"context"
)

var userClient userservice.Client

func Register(ctx context.Context, req *user.RegisterReq) error {
	resp, err := userClient.Register(ctx, req)
	if err != nil {
		return err
	}

	response := resp.Resp
	if response.Code != 0 {
		return errno.NewErrNo(response.Code, response.Message)
	}
	return nil
}

func Login(ctx context.Context, req *user.LoginReq) (*user.LoginResp, error) {
	resp, err := userClient.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}

func UploadResume(ctx context.Context, req *user.UploadResumeReq) (*user.UploadResumeResp, error) {
	resp, err := userClient.UploadResume(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}
