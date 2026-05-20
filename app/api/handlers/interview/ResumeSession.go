package interview

import (
	"Goffer/app/api/handlers/pack"
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/interview"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

type ResumeParam struct {
	SessionID string `json:"session_id"`
}

func ResumeSession(ctx context.Context, c *app.RequestContext) {
	var resumeVar ResumeParam
	if err := c.Bind(&resumeVar); err != nil {
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	if resumeVar.SessionID == "" {
		pack.SendResponse(c, errno.ParamErr, nil)
		return
	}

	logger.InfoCtx(ctx, "恢复会话请求", zap.String("session_id", resumeVar.SessionID))

	resp, err := rpc.ResumeSession(ctx, &interview.ResumeSessionReq{
		SessionId: resumeVar.SessionID,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "恢复会话失败",
			zap.String("session_id", resumeVar.SessionID),
			zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, map[string]interface{}{
		"fsm_state": resp.FsmState,
		"round":     resp.Round,
		"history":   resp.History,
	})
}
