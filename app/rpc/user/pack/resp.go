package pack

import (
	"Goffer/kitex_gen/base"
	"Goffer/pkg/errno"
	"errors"

	"github.com/cloudwego/kitex/pkg/klog"
)

// BuildBaseResp build baseResp from error
func BuildBaseResp(err error) *base.Response {
	if err == nil {
		return baseResp(errno.Success)
	}

	e := errno.ErrNo{}
	// 如果是业务定义的 ErrNo（包括被 fmt.Errorf %w 包装过的），正常返回给前端
	if errors.As(err, &e) {
		return baseResp(e)
	}
	// 非业务错误，记录日志，并对前端隐藏细节
	klog.Errorf("Internal Server Error: %v", err)

	return baseResp(errno.ServiceErr)
}

func baseResp(err errno.ErrNo) *base.Response {
	return &base.Response{Code: err.ErrCode, Message: err.ErrMsg}
}
