package user

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/user"
	context2 "Goffer/pkg/contextutil"
	"Goffer/pkg/errno"
	"context"
	"io"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func UploadResume(ctx context.Context, c *app.RequestContext) {
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

	if fileHeader.Size > 10<<20 {
		SendResponse(c, errno.FileUploadErr.WithMessage("File size exceeds 10MB limit"), nil)
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

	contentType := http.DetectContentType(fileContent)
	if contentType != "application/pdf" && contentType != "image/jpeg" && contentType != "image/png" {
		SendResponse(c, errno.FileFormatErr, nil)
		return
	}

	resp, err := rpc.UploadResume(ctx, &user.UploadResumeReq{
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
		"resumeID": resp.ResumeId,
		"fileURL":  resp.FileUrl,
	})
}
