package router

import (
	"Goffer/app/api/handlers/interview"
	"Goffer/app/api/handlers/user"
	"Goffer/pkg/jwt"
	"Goffer/pkg/middleware"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func InitRouter(h *server.Hertz, jwtManager *jwt.JWTManager) {

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
	}

}
