package user

import (
	"Goffer/app/api/handlers/pack"
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

func Register(ctx context.Context, c *app.RequestContext) {
	var registerVar UserParam
	if err := c.Bind(&registerVar); err != nil {
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	if len(registerVar.UserName) == 0 || len(registerVar.PassWord) == 0 {
		pack.SendResponse(c, errno.ParamErr, nil)
		return
	}

	logger.InfoCtx(ctx, "用户注册请求", zap.String("username", registerVar.UserName))

	err := rpc.Register(ctx, &user.RegisterReq{
		Username: registerVar.UserName,
		Password: registerVar.PassWord,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "注册失败", zap.String("username", registerVar.UserName), zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, nil)
}
