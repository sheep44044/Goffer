package rpc

import (
	"Goffer/kitex_gen/knowledge"
	"Goffer/kitex_gen/knowledge/knowledgeservice"
	"Goffer/pkg/errno"
	"context"
)

var knowledgeClient knowledgeservice.Client

func IngestQuestion(ctx context.Context, req *knowledge.IngestQuestionReq) (*knowledge.IngestQuestionResp, error) {
	resp, err := knowledgeClient.IngestQuestion(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}

func UploadQuestion(ctx context.Context, req *knowledge.UploadQuestionReq) (*knowledge.UploadQuestionResp, error) {
	resp, err := knowledgeClient.UploadQuestion(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}

func IngestJD(ctx context.Context, req *knowledge.IngestJDReq) (*knowledge.IngestJDResp, error) {
	resp, err := knowledgeClient.IngestJD(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}

func UploadJD(ctx context.Context, req *knowledge.UploadJDReq) (*knowledge.UploadJDResp, error) {
	resp, err := knowledgeClient.UploadJD(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}
