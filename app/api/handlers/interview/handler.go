package interview

type StartParam struct {
	ResumeId string `json:"resume_id"`
}

type ChatReq struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}
