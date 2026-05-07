package interview

import (
	"Goffer/app/rpc/interview/service"
	"Goffer/app/rpc/interview/svc"
	"Goffer/app/rpc/user/pack"
	"fmt"

	"Goffer/kitex_gen/interview"
	"Goffer/pkg/errno"
	"context"
)

type interviewServiceImpl struct {
	svc *svc.ServiceContext
}

func (s interviewServiceImpl) ChatStream(req *interview.ChatReq, stream interview.InterviewService_ChatStreamServer) (err error) {
	ctx := stream.Context()

	if req.SessionId == "" {
		return fmt.Errorf("session_id is required")
	}
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}

	return service.NewChatService(s.svc).ChatStream(ctx, req, stream)
}

func (s interviewServiceImpl) StartInterview(ctx context.Context, req *interview.StartInterviewReq) (resp *interview.StartInterviewResp, err error) {
	resp = new(interview.StartInterviewResp)

	if len(req.UserId) == 0 || len(req.ResumeId) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	resp, err = service.NewStartService(s.svc).StartInterview(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(errno.ServiceErr)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	return resp, nil
}

func (s interviewServiceImpl) GetChatContext(ctx context.Context, req *interview.GetChatContextReq) (resp *interview.GetChatContextResp, err error) {
	resp = new(interview.GetChatContextResp)

	if len(req.SessionId) == 0 || len(req.LatestUserMsg) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	resp, err = service.NewGetChatService(s.svc).GetChatContextInterview(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(errno.ServiceErr)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	return resp, nil
}
