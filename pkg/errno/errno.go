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

	// 新增业务线错误码：文件与解析相关
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

func ConvertErr(err error) ErrNo {
	Err := ErrNo{}
	if errors.As(err, &Err) {
		return Err
	}

	s := ServiceErr
	s.ErrMsg = err.Error()
	return s
}
