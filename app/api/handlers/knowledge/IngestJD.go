package knowledge

import (
	"Goffer/app/api/handlers/pack"
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

func IngestJD(ctx context.Context, c *app.RequestContext) {
	var req JDVar
	if err := c.BindAndValidate(&req); err != nil {
		pack.SendResponse(c, errno.ParamErr.WithMessage(err.Error()), nil)
		return
	}

	logger.InfoCtx(ctx, "录入 JD 请求",
		zap.String("company", req.Company),
		zap.String("title", req.Title))

	resp, err := rpc.IngestJD(ctx, &knowledge.IngestJDReq{
		Company:          req.Company,
		Title:            req.Title,
		Responsibilities: req.Responsibilities,
		Requirements:     req.Requirements,
		Tags:             req.Tags,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "录入 JD 失败",
			zap.String("company", req.Company),
			zap.String("title", req.Title),
			zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, resp)
}
