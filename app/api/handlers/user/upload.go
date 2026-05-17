package user

import (
	"Goffer/app/api/handlers/pack"
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/contextutil"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"
	"io"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"go.uber.org/zap"
)

func UploadResume(ctx context.Context, c *app.RequestContext) {
	userID, err := contextutil.GetUserIDFromGateway(c)
	if err != nil {
		pack.SendResponse(c, errno.AuthorizationFailedErr, nil)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		pack.SendResponse(c, errno.FileUploadErr.WithMessage("Failed to get file from request"), nil)
		return
	}

	if fileHeader.Size > 10<<20 {
		pack.SendResponse(c, errno.FileUploadErr.WithMessage("File size exceeds 10MB limit"), nil)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		pack.SendResponse(c, errno.FileUploadErr.WithMessage("Failed to open uploaded file"), nil)
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorCtx(ctx, "读取上传文件失败", zap.String("file_name", fileHeader.Filename), zap.Error(err))
		pack.SendResponse(c, errno.FileUploadErr.WithMessage("Failed to read file content"), nil)
		return
	}

	contentType := http.DetectContentType(fileContent)
	if contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" {
		pack.SendResponse(c, errno.FileFormatErr, nil)
		return
	}

	logger.InfoCtx(ctx, "上传简历请求",
		zap.String("user_id", userID),
		zap.String("file_name", fileHeader.Filename),
		zap.Int64("file_size", fileHeader.Size))

	resp, err := rpc.UploadResume(ctx, &user.UploadResumeReq{
		UserId:      userID,
		FileName:    fileHeader.Filename,
		FileContent: fileContent,
		ContentType: contentType,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "上传简历RPC调用失败",
			zap.String("user_id", userID),
			zap.String("file_name", fileHeader.Filename),
			zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, utils.H{
		"resumeID": resp.ResumeId,
		"fileURL":  resp.FileUrl,
	})
}
