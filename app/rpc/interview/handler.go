package interview

import (
	"Goffer/app/rpc/interview/service"
	"Goffer/app/rpc/interview/svc"
	"Goffer/app/rpc/user/pack"

	"Goffer/kitex_gen/interview"
	"Goffer/pkg/errno"
	"context"
)

type interviewServiceImpl struct {
	svc *svc.ServiceContext
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

func (s interviewServiceImpl) SaveChatRecord(ctx context.Context, req *interview.SaveChatRecordReq) (resp *interview.SaveChatRecordResp, err error) {
	resp = new(interview.SaveChatRecordResp)

	if len(req.SessionId) == 0 || len(req.UserMsg) == 0 || len(req.AiMsg) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	err = service.NewSaveChatService(s.svc).SaveChatRecordInterview(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(errno.ServiceErr)
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	return resp, nil
}
