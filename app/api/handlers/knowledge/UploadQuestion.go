package knowledge

import (
	"Goffer/app/api/handlers/pack"
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/contextutil"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"go.uber.org/zap"
)

func UploadQuestion(ctx context.Context, c *app.RequestContext) {
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

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".csv" {
		pack.SendResponse(c, errno.FileFormatErr.WithMessage("Only CSV files are allowed"), nil)
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
		logger.ErrorCtx(ctx, "读取题库上传文件失败", zap.String("file_name", fileHeader.Filename), zap.Error(err))
		pack.SendResponse(c, errno.FileUploadErr.WithMessage("Failed to read file content"), nil)
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/csv"
	}

	logger.InfoCtx(ctx, "上传题库文件请求",
		zap.String("user_id", userID),
		zap.String("file_name", fileHeader.Filename),
		zap.Int64("file_size", fileHeader.Size))

	resp, err := rpc.UploadQuestion(ctx, &knowledge.UploadQuestionReq{
		UserId:      userID,
		FileName:    fileHeader.Filename,
		FileContent: fileContent,
		ContentType: contentType,
	})
	if err != nil {
		logger.ErrorCtx(ctx, "上传题库文件 RPC 调用失败",
			zap.String("user_id", userID),
			zap.String("file_name", fileHeader.Filename),
			zap.Error(err))
		pack.SendResponse(c, errno.ConvertErr(err), nil)
		return
	}

	pack.SendResponse(c, errno.Success, utils.H{
		"taskID":  resp.TaskId,
		"fileURL": resp.FileUrl,
	})
}
