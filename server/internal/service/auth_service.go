// server/internal/service/auth_service.go
package service

import (
	"context"

	"server/internal/biz"
)

type AuthService struct {
	uc *biz.AuthUsecase
}

func NewAuthService(uc *biz.AuthUsecase) *AuthService {
	return &AuthService{uc: uc}
}

// JSON-RPC: auth.register
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthReply struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"` // Unix 时间戳
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthReply, error) {
	token, expiresAt, user, err := s.uc.Register(ctx, req.Username, req.Password)
	if err != nil {
		// 这里你可以转成 JSON-RPC error（比如用 code 映射）
		return nil, err
	}
	return &AuthReply{
		UserID:      user.ID,
		Username:    user.Username,
		AccessToken: token,
		ExpiresAt:   expiresAt.Unix(),
	}, nil
}

// JSON-RPC: auth.login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*AuthReply, error) {
	token, expiresAt, user, err := s.uc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthReply{
		UserID:      user.ID,
		Username:    user.Username,
		AccessToken: token,
		ExpiresAt:   expiresAt.Unix(),
	}, nil
}
