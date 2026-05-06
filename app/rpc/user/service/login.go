package service

import (
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	svc *svc.ServiceContext
}

func NewLoginService(svc *svc.ServiceContext) *LoginService {
	return &LoginService{
		svc: svc,
	}
}

func (s *LoginService) Login(ctx context.Context, req *user.LoginReq) (string, error) {
	users, err := s.svc.DB.QueryUser(ctx, req.Username)
	if err != nil {
		return "", fmt.Errorf("query user from db failed: %w", err)
	}
	if len(users) == 0 {
		return "", errno.UserNotExistErr
	}

	if err := bcrypt.CompareHashAndPassword([]byte(users[0].Password), []byte(req.Password)); err != nil {
		return "", errno.PasswordMismatchErr
	}

	userId, username := users[0].ID, users[0].Username
	token, err := s.svc.JWT.GenerateToken(userId, username, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("generate jwt token failed: %w", err)
	}

	return token, nil
}
