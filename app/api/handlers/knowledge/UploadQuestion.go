package knowledge

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/knowledge"
	context2 "Goffer/pkg/contextutil"
	"Goffer/pkg/errno"
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func UploadQuestion(ctx context.Context, c *app.RequestContext) {
	userID, err := context2.GetUserIDFromGateway(c)
	if err != nil {
		SendResponse(c, errno.AuthorizationFailedErr, nil)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		SendResponse(c, errno.FileUploadErr.WithMessage("Failed to get file from request"), nil)
		return
	}

	// 2. 文件大小限制：如果 CSV 包含大量数据，可能需要调整这个 10MB 的限制
	if fileHeader.Size > 10<<20 {
		SendResponse(c, errno.FileUploadErr.WithMessage("File size exceeds 10MB limit"), nil)
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".csv" {
		SendResponse(c, errno.FileFormatErr.WithMessage("Only CSV files are allowed"), nil)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		SendResponse(c, errno.FileUploadErr.WithMessage("Failed to open uploaded file"), nil)
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		SendResponse(c, errno.FileUploadErr.WithMessage("Failed to read file content"), nil)
		return
	}

	// 可选：如果你需要获取客户端传递的 Content-Type 进行记录
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/csv" // 给个默认值
	}

	resp, err := rpc.UploadQuestion(ctx, &knowledge.UploadQuestionReq{
		UserId:      userID,
		FileName:    fileHeader.Filename,
		FileContent: fileContent,
		ContentType: contentType,
	})
	if err != nil {
		SendResponse(c, errno.ConvertErr(err), nil)
		return
	}
	SendResponse(c, errno.Success, utils.H{
		"taskID":  resp.TaskId,
		"fileURL": resp.FileUrl,
	})
}
