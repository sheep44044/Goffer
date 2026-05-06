package service

import (
	"Goffer/app/rpc/user/dal/db"
	"Goffer/app/rpc/user/svc"
	"Goffer/kitex_gen/user"
	"Goffer/pkg/errno"
	"Goffer/pkg/snowflake"
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type RegisterService struct {
	svc *svc.ServiceContext
}

// NewRegisterService new CreateUserService
func NewRegisterService(svc *svc.ServiceContext) *RegisterService {
	return &RegisterService{svc: svc}
}

// Register create user info.
func (s *RegisterService) Register(ctx context.Context, req *user.RegisterReq) error {
	users, err := s.svc.DB.QueryUser(ctx, req.Username)
	if err != nil {
		return fmt.Errorf("query user from db failed: %w", err)
	}
	if len(users) != 0 {
		return errno.UserAlreadyExistErr
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate password failed: %w", err)
	}

	newUserID := snowflake.GenString()
	return s.svc.DB.CreateUser(ctx, []*db.User{{
		ID:       newUserID,
		Username: req.Username,
		Password: string(hashed),
	}})
}
