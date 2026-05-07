package user

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func Register(ctx context.Context, c *app.RequestContext) {
	var registerVar UserParam
	if err := c.Bind(&registerVar); err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	if len(registerVar.UserName) == 0 || len(registerVar.PassWord) == 0 {
		SendResponse(c, errno.ParamErr, nil)
		return
	}

	err := rpc.Register(ctx, &user.RegisterReq{
		Username: registerVar.UserName,
		Password: registerVar.PassWord,
	})
	if err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	SendResponse(c, errno.Success, nil)
}
