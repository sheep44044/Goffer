package interview

import (
	"Goffer/app/rpc/interview/service"
	"Goffer/app/rpc/interview/svc"
	"Goffer/app/rpc/user/pack"
	"Goffer/kitex_gen/interview"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"go.uber.org/zap"
)

type interviewServiceImpl struct {
	svc *svc.ServiceContext
}

func (s interviewServiceImpl) ChatStream(req *interview.ChatReq, stream interview.InterviewService_ChatStreamServer) (err error) {
	ctx := stream.Context()

	if req.SessionId == "" || req.Message == "" {
		logger.Error("面试流式请求参数缺失",
			zap.String("session_id", req.SessionId),
			zap.Bool("has_message", req.Message != ""))
		return errno.ParamErr
	}

	logger.InfoCtx(ctx, "RPC 流式对话请求",
		zap.String("session_id", req.SessionId),
		zap.Int("msg_len", len(req.Message)))

	return service.NewChatService(s.svc).ChatStream(ctx, req, stream)
}

func (s interviewServiceImpl) StartInterview(ctx context.Context, req *interview.StartInterviewReq) (resp *interview.StartInterviewResp, err error) {
	resp = new(interview.StartInterviewResp)

	if len(req.UserId) == 0 || len(req.ResumeId) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 开始面试请求",
		zap.String("user_id", req.UserId),
		zap.String("resume_id", req.ResumeId))

	resp, err = service.NewStartService(s.svc).StartInterview(ctx, req)
	if err != nil {
		// 修复：使用 err 本身而非 errno.ServiceErr，保留业务错误码
		logger.ErrorCtx(ctx, "开始面试失败",
			zap.String("user_id", req.UserId),
			zap.String("resume_id", req.ResumeId),
			zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
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
		// 修复：使用 err 本身而非 errno.ServiceErr
		logger.ErrorCtx(ctx, "获取聊天上下文失败",
			zap.String("session_id", req.SessionId),
			zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	return resp, nil
}
