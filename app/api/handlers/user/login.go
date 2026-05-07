package user

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func Login(ctx context.Context, c *app.RequestContext) {
	var loginVar UserParam
	if err := c.Bind(&loginVar); err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	if len(loginVar.UserName) == 0 || len(loginVar.PassWord) == 0 {
		SendResponse(c, errno.ParamErr, nil)
		return
	}

	resp, err := rpc.Login(ctx, &user.LoginReq{
		Username: loginVar.UserName,
		Password: loginVar.PassWord,
	})
	if err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	SendResponse(c, errno.Success, resp.Token)
}
