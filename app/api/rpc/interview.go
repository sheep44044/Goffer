package rpc

import (
	"Goffer/kitex_gen/interview"
	"Goffer/kitex_gen/interview/interviewservice"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"github.com/cloudwego/kitex/client/callopt/streamcall"
	"go.uber.org/zap"
)

var (
	interviewClient       interviewservice.Client
	interviewStreamClient interviewservice.StreamClient
)

func StartInterview(ctx context.Context, req *interview.StartInterviewReq) (*interview.StartInterviewResp, error) {
	resp, err := interviewClient.StartInterview(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 Interview.StartInterview 失败",
			zap.String("user_id", req.UserId),
			zap.String("resume_id", req.ResumeId),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}

func ChatStream(ctx context.Context, req *interview.ChatReq, opts ...streamcall.Option) (interviewservice.InterviewService_ChatStreamClient, error) {
	return interviewStreamClient.ChatStream(ctx, req, opts...)
}

func ResumeSession(ctx context.Context, req *interview.ResumeSessionReq) (*interview.ResumeSessionResp, error) {
	resp, err := interviewClient.ResumeSession(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 Interview.ResumeSession 失败",
			zap.String("session_id", req.SessionId),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}
