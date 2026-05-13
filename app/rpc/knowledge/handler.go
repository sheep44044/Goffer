package knowledge

import (
	"Goffer/app/rpc/knowledge/service"
	"Goffer/app/rpc/knowledge/svc"
	"Goffer/app/rpc/user/pack"
	"Goffer/kitex_gen/knowledge"
	"Goffer/pkg/errno"
	"context"
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

	questionID, err := service.NewQuestionService(s.svc).IngestQuestion(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
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

	csvID, fileURL, err := service.NewQuestionCSVService(s.svc).UploadQuestionCSV(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
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

	JDid, err := service.NewJDService(s.svc).IngestQuestion(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
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

	csvID, fileURL, err := service.NewJDCSVService(s.svc).UploadJDCSV(ctx, req)
	if err != nil {
		resp.Resp = pack.BuildBaseResp(err)
		return resp, nil
	}

	resp.Resp = pack.BuildBaseResp(errno.Success)
	resp.TaskId = csvID
	resp.FileUrl = fileURL
	return resp, nil
}
