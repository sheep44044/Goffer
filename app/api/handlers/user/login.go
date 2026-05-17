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

func Login(ctx context.Context, c *app.RequestContext) {
	var loginVar UserParam
	if err := c.Bind(&loginVar); err != nil {
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	if len(loginVar.UserName) == 0 || len(loginVar.PassWord) == 0 {
		pack.SendResponse(c, errno.ParamErr, nil)
		return
	}

	logger.InfoCtx(ctx, "用户登录请求", zap.String("username", loginVar.UserName))

	resp, err := rpc.Login(ctx, &user.LoginReq{
		Username: loginVar.UserName,
		Password: loginVar.PassWord,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "登录失败", zap.String("username", loginVar.UserName), zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, resp.Token)
}
