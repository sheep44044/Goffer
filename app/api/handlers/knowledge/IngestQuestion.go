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

func IngestQuestion(ctx context.Context, c *app.RequestContext) {
	var req QuestionVar
	if err := c.BindAndValidate(&req); err != nil {
		pack.SendResponse(c, errno.ParamErr.WithMessage(err.Error()), nil)
		return
	}

	logger.InfoCtx(ctx, "录入题目请求",
		zap.Int("content_len", len(req.QuestionContent)),
		zap.Strings("tags", req.Tags))

	resp, err := rpc.IngestQuestion(ctx, &knowledge.IngestQuestionReq{
		QuestionContent: req.QuestionContent,
		StandardAnswer:  req.StandardAnswer,
		Tags:            req.Tags,
		Difficulty:      req.Difficulty,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "录入题目失败",
			zap.Int("content_len", len(req.QuestionContent)),
			zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, resp)
}
