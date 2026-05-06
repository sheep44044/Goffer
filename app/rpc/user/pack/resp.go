package pack

import (
	"Goffer/kitex_gen/base"
	"Goffer/pkg/errno"
	"errors"
)

// BuildBaseResp build baseResp from error
func BuildBaseResp(err error) *base.Response {
	if err == nil {
		return baseResp(errno.Success)
	}

	e := errno.ErrNo{}
	if errors.As(err, &e) {
		return baseResp(e)
	}

	s := errno.ServiceErr.WithMessage(err.Error())
	return baseResp(s)
}

func baseResp(err errno.ErrNo) *base.Response {
	return &base.Response{Code: err.ErrCode, Message: err.ErrMsg}
}
