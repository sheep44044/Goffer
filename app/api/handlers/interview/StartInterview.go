package interview

import (
	"Goffer/app/api/handlers/pack"
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/interview"
	"Goffer/pkg/contextutil"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"go.uber.org/zap"
)

func StartInterview(ctx context.Context, c *app.RequestContext) {
	var startVar StartParam
	if err := c.Bind(&startVar); err != nil {
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	userID, err := contextutil.GetUserIDFromGateway(c)
	if err != nil {
		pack.SendResponse(c, errno.AuthorizationFailedErr, nil)
		return
	}

	if len(userID) == 0 {
		pack.SendResponse(c, errno.ParamErr, nil)
		return
	}

	logger.InfoCtx(ctx, "开始面试请求",
		zap.String("user_id", userID),
		zap.String("resume_id", startVar.ResumeId))

	resp, err := rpc.StartInterview(ctx, &interview.StartInterviewReq{
		UserId:   userID,
		ResumeId: startVar.ResumeId,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "开始面试失败",
			zap.String("user_id", userID),
			zap.String("resume_id", startVar.ResumeId),
			zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, utils.H{
		"session_id":     resp.SessionId,
		"opening_remark": resp.OpeningRemark,
	})
}
