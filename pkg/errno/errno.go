package errno

import (
	"errors"
	"fmt"
)

const (
	SuccessCode                = 0
	ServiceErrCode             = 10001
	ParamErrCode               = 10002
	UserAlreadyExistErrCode    = 10003
	AuthorizationFailedErrCode = 10004
	UserNotExistErrCode        = 10005
	PasswordMismatchErrCode    = 10006

	FileUploadErrCode     = 20001
	FileFormatErrCode     = 20002
	ResumeParseErrCode    = 20003
	ResumeNotFoundErrCode = 20004
)

type ErrNo struct {
	ErrCode int64
	ErrMsg  string
}

func (e ErrNo) Error() string {
	return fmt.Sprintf("err_code=%d, err_msg=%s", e.ErrCode, e.ErrMsg)
}

func NewErrNo(code int64, msg string) ErrNo {
	return ErrNo{code, msg}
}

func (e ErrNo) WithMessage(msg string) ErrNo {
	e.ErrMsg = msg
	return e
}

var (
	Success                = NewErrNo(SuccessCode, "Success")
	ServiceErr             = NewErrNo(ServiceErrCode, "Service is unable to start successfully")
	ParamErr               = NewErrNo(ParamErrCode, "Wrong Parameter has been given")
	UserAlreadyExistErr    = NewErrNo(UserAlreadyExistErrCode, "User already exists")
	AuthorizationFailedErr = NewErrNo(AuthorizationFailedErrCode, "Authorization failed")
	UserNotExistErr        = NewErrNo(UserNotExistErrCode, "User does not exist")
	PasswordMismatchErr    = NewErrNo(PasswordMismatchErrCode, "Password mismatch")

	FileUploadErr     = NewErrNo(FileUploadErrCode, "Failed to upload file")
	FileFormatErr     = NewErrNo(FileFormatErrCode, "Unsupported file format")
	ResumeParseErr    = NewErrNo(ResumeParseErrCode, "AI failed to parse resume")
	ResumeNotFoundErr = NewErrNo(ResumeNotFoundErrCode, "Resume does not exist")
)

// ConvertErr 将任意 error 转换为 ErrNo。
// 若 err 本身是 ErrNo 或被 %w 包装过，保留业务错误码与消息；
// 否则返回 ServiceErr，且不暴露底层技术错误细节给前端。
func ConvertErr(err error) ErrNo {
	var e ErrNo
	if errors.As(err, &e) {
		return e
	}
	return ServiceErr
}

// IsBizErr 判断 err 或其包装链中是否包含业务定义的 ErrNo。
// 用于在日志层面区分业务预期错误和需要告警的系统异常。
func IsBizErr(err error) bool {
	var e ErrNo
	return errors.As(err, &e)
}
