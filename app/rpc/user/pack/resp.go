package pack

import (
	"Goffer/kitex_gen/base"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"
	"errors"

	"go.uber.org/zap"
)

// BuildBaseResp 将 error 转换为 *base.Response。
func BuildBaseResp(err error) *base.Response {
	if err == nil {
		return resp(errno.Success)
	}
	var e errno.ErrNo
	if errors.As(err, &e) {
		return resp(e)
	}
	logger.Error("RPC Internal Server Error", zap.Error(err))
	return resp(errno.ServiceErr)
}

// BuildBaseRespCtx 同 BuildBaseResp，但日志携带 ctx 中的 OTel TraceID。
func BuildBaseRespCtx(ctx context.Context, err error) *base.Response {
	if err == nil {
		return resp(errno.Success)
	}
	var e errno.ErrNo
	if errors.As(err, &e) {
		return resp(e)
	}
	logger.ErrorCtx(ctx, "RPC Internal Server Error", zap.Error(err))
	return resp(errno.ServiceErr)
}

func resp(e errno.ErrNo) *base.Response {
	return &base.Response{Code: e.ErrCode, Message: e.ErrMsg}
}
