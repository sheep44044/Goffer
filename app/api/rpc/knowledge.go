package rpc

import (
	"Goffer/kitex_gen/knowledge"
	"Goffer/kitex_gen/knowledge/knowledgeservice"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"go.uber.org/zap"
)

var knowledgeClient knowledgeservice.Client

func IngestQuestion(ctx context.Context, req *knowledge.IngestQuestionReq) (*knowledge.IngestQuestionResp, error) {
	resp, err := knowledgeClient.IngestQuestion(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 Knowledge.IngestQuestion 失败",
			zap.Int("content_len", len(req.QuestionContent)),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}

func UploadQuestion(ctx context.Context, req *knowledge.UploadQuestionReq) (*knowledge.UploadQuestionResp, error) {
	resp, err := knowledgeClient.UploadQuestion(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 Knowledge.UploadQuestion 失败",
			zap.String("user_id", req.UserId),
			zap.String("file_name", req.FileName),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}

func IngestJD(ctx context.Context, req *knowledge.IngestJDReq) (*knowledge.IngestJDResp, error) {
	resp, err := knowledgeClient.IngestJD(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 Knowledge.IngestJD 失败",
			zap.String("company", req.Company),
			zap.String("title", req.Title),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}

func UploadJD(ctx context.Context, req *knowledge.UploadJDReq) (*knowledge.UploadJDResp, error) {
	resp, err := knowledgeClient.UploadJD(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "调用 Knowledge.UploadJD 失败",
			zap.String("user_id", req.UserId),
			zap.String("file_name", req.FileName),
			zap.Error(err))
		return nil, err
	}

	if resp.Resp.Code != 0 {
		return nil, errno.NewErrNo(resp.Resp.Code, resp.Resp.Message)
	}
	return resp, nil
}
