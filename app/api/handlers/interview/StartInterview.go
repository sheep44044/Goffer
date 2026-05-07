package interview

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/interview"
	context2 "Goffer/pkg/contextutil"
	"Goffer/pkg/errno"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func StartInterview(ctx context.Context, c *app.RequestContext) {
	var startVar StartParam
	if err := c.Bind(&startVar); err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	userID, err := context2.GetUserIDFromGateway(c)
	if err != nil {
		SendResponse(c, errno.AuthorizationFailedErr, nil)
		return
	}

	if len(userID) == 0 {
		SendResponse(c, errno.ParamErr, nil)
		return
	}

	resp, err := rpc.StartInterview(ctx, &interview.StartInterviewReq{
		UserId:   userID,
		ResumeId: startVar.ResumeId,
	})
	if err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	SendResponse(c, errno.Success, resp.OpeningRemark)
}
