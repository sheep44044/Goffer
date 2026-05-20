package router

import (
	"Goffer/app/api/handlers/interview"
	"Goffer/app/api/handlers/knowledge"
	"Goffer/app/api/handlers/user"
	"Goffer/pkg/jwt"
	"Goffer/pkg/middleware"

	"github.com/cloudwego/hertz/pkg/app/server"
	"golang.org/x/time/rate"
)

func InitRouter(h *server.Hertz, jwtManager *jwt.JWTManager) {
	h.Use(middleware.GlobalRateLimitMiddleware(rate.Limit(100), 200))

	publicGroup := h.Group("/api/user")
	{
		publicGroup.POST("/register", user.Register)
		publicGroup.POST("/login", user.Login)
	}

	authGroup := h.Group("/api/resume")
	authGroup.Use(middleware.JWTAuthMiddleware(jwtManager))
	{
		authGroup.POST("/upload", user.UploadResume)
	}

	interviewGroup := h.Group("/api/interview")
	interviewGroup.Use(middleware.JWTAuthMiddleware(jwtManager))
	{
		interviewGroup.POST("/start", interview.StartInterview)
		interviewGroup.POST("/chat", interview.ChatStream)
		interviewGroup.POST("/resume", interview.ResumeSession)
	}

	knowledgeGroup := h.Group("/api/knowledge")
	knowledgeGroup.Use(middleware.JWTAuthMiddleware(jwtManager))
	{
		// 岗位 JD 管理
		knowledgeGroup.POST("/jd/upload", knowledge.UploadJD)
		knowledgeGroup.POST("/jd/ingest", knowledge.IngestJD)

		// 题库管理
		knowledgeGroup.POST("/question/upload", knowledge.UploadQuestion)
		knowledgeGroup.POST("/question/ingest", knowledge.IngestQuestion)
	}

}
