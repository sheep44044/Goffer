package main

import (
	"Goffer/app/rpc/knowledge/service"
	"Goffer/app/rpc/knowledge/svc"
	"Goffer/app/rpc/user/pack"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"Goffer/pkg/logger"
	"context"

	"go.uber.org/zap"
)

type knowledgeServiceImpl struct {
	svc *svc.ServiceContext
}

func (s *knowledgeServiceImpl) IngestQuestion(ctx context.Context, req *knowledge.IngestQuestionReq) (resp *knowledge.IngestQuestionResp, err error) {
	resp = new(knowledge.IngestQuestionResp)
	if len(req.QuestionContent) == 0 || len(req.StandardAnswer) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 录入题目请求", zap.Int("content_len", len(req.QuestionContent)))

	questionID, err := service.NewQuestionService(s.svc).IngestQuestion(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "录入题目失败", zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}
	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.QuestionId = questionID
	return resp, nil
}

func (s *knowledgeServiceImpl) UploadQuestion(ctx context.Context, req *knowledge.UploadQuestionReq) (resp *knowledge.UploadQuestionResp, err error) {
	resp = new(knowledge.UploadQuestionResp)
	if len(req.UserId) == 0 || len(req.FileContent) == 0 || len(req.FileName) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 上传题库文件请求",
		zap.String("user_id", req.UserId),
		zap.String("file_name", req.FileName))

	csvID, fileURL, err := service.NewQuestionCSVService(s.svc).UploadQuestionCSV(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "上传题库文件失败", zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.TaskId = csvID
	resp.FileUrl = fileURL
	return resp, nil
}

func (s *knowledgeServiceImpl) IngestJD(ctx context.Context, req *knowledge.IngestJDReq) (resp *knowledge.IngestJDResp, err error) {
	resp = new(knowledge.IngestJDResp)
	if len(req.Title) == 0 || len(req.Company) == 0 || len(req.Responsibilities) == 0 || len(req.Requirements) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 录入 JD 请求",
		zap.String("company", req.Company),
		zap.String("title", req.Title))

	JDid, err := service.NewJDService(s.svc).IngestQuestion(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "录入 JD 失败", zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.JdId = JDid
	return resp, nil
}

func (s *knowledgeServiceImpl) UploadJD(ctx context.Context, req *knowledge.UploadJDReq) (resp *knowledge.UploadJDResp, err error) {
	resp = new(knowledge.UploadJDResp)
	if len(req.UserId) == 0 || len(req.FileContent) == 0 || len(req.FileName) == 0 {
		resp.Resp = pack.BuildBaseResp(errno.ParamErr)
		return resp, nil
	}

	logger.InfoCtx(ctx, "RPC 上传 JD 文件请求",
		zap.String("user_id", req.UserId),
		zap.String("file_name", req.FileName))

	csvID, fileURL, err := service.NewJDCSVService(s.svc).UploadJDCSV(ctx, req)
	if err != nil {
		logger.ErrorCtx(ctx, "上传 JD 文件失败", zap.Error(err))
		resp.Resp = pack.BuildBaseRespCtx(ctx, err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.TaskId = csvID
	resp.FileUrl = fileURL
	return resp, nil
}
