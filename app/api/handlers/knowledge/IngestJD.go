package knowledge

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func IngestJD(ctx context.Context, c *app.RequestContext) {
	var req JDVar
	if err := c.BindAndValidate(&req); err != nil {
		SendResponse(c, errno.ParamErr.WithMessage(err.Error()), nil)
		return
	}

	resp, err := rpc.IngestJD(ctx, &knowledge.IngestJDReq{
		Company:          req.Company,
		Title:            req.Title,
		Responsibilities: req.Responsibilities,
		Requirements:     req.Requirements,
		Tags:             req.Tags,
	})
	if err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	SendResponse(c, errno.Success, resp)
}
