package knowledge

import (
	"Goffer/pkg/errno"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type Response struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// SendResponse pack response
func SendResponse(c *app.RequestContext, err error, data interface{}) {
	Err := errno.ConvertErr(err)
	c.JSON(consts.StatusOK, Response{
		Code:    Err.ErrCode,
		Message: Err.ErrMsg,
		Data:    data,
	})
}

type QuestionVar struct {
	QuestionContent string   `json:"question_content" vd:"$!=''; msg:'题目内容不能为空'"`
	StandardAnswer  string   `json:"standard_answer" vd:"$!=''; msg:'标准答案不能为空'"`
	Tags            []string `json:"tags"`
	Difficulty      *string  `json:"difficulty"`
}

type JDVar struct {
	Company          string   `json:"company" vd:"$!=''; msg:'公司名称不能为空'"`
	Title            string   `json:"title" vd:"$!=''; msg:'职位标题不能为空'"`
	Responsibilities string   `json:"responsibilities" vd:"$!=''; msg:'岗位职责不能为空'"`
	Requirements     string   `json:"requirements" vd:"$!=''; msg:'任职要求不能为空'"`
	Tags             []string `json:"tags"`
}
