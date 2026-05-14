package knowledge

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func IngestQuestion(ctx context.Context, c *app.RequestContext) {
	var req QuestionVar
	if err := c.BindAndValidate(&req); err != nil {
		// 如果前端没传必填字段，这里会自动拦截并返回 msg
		SendResponse(c, errno.ParamErr.WithMessage(err.Error()), nil)
		return
	}

	resp, err := rpc.IngestQuestion(ctx, &knowledge.IngestQuestionReq{
		QuestionContent: req.QuestionContent,
		StandardAnswer:  req.StandardAnswer,
		Tags:            req.Tags,
		Difficulty:      req.Difficulty,
	})
	if err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	SendResponse(c, errno.Success, resp)
}
