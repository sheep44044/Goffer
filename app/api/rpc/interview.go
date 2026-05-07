package rpc

import (
	"Goffer/kitex_gen/interview"
	"Goffer/kitex_gen/interview/interviewservice"
	"Goffer/pkg/errno"
	"context"
)

var (
	interviewClient       interviewservice.Client
	interviewStreamClient interviewservice.StreamClient
)

func StartInterview(ctx context.Context, req *interview.StartInterviewReq) (*interview.StartInterviewResp, error) {
	resp, err := interviewClient.StartInterview(ctx, req)
	if err != nil {
		return nil, err
	}

	response := resp.Resp
	if response.Code != 0 {
		return nil, errno.NewErrNo(response.Code, response.Message)
	}
	return resp, nil
}

func ChatStream(ctx context.Context, req *interview.ChatReq) (interviewservice.InterviewService_ChatStreamClient, error) {
	return interviewStreamClient.ChatStream(ctx, req)
}
