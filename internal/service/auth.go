package service

import (
	"context"
	"gaap-api/internal/model"
)

type IAuth interface {
	Login(ctx context.Context, in model.LoginInput) (out *model.AuthResponse, err error)
	Register(ctx context.Context, in model.RegisterInput) (out *model.AuthResponse, err error)
	RefreshToken(ctx context.Context, refreshToken string) (out *model.TokenPair, err error)
	Generate2FA(ctx context.Context) (out *model.TwoFactorSecret, err error)
	Enable2FA(ctx context.Context, code string) (err error)
	Disable2FA(ctx context.Context, code string, password string) (err error)
	AddTokenToBlacklist(ctx context.Context, token string)
	IsTokenBlacklisted(ctx context.Context, token string) bool
}

var localAuth IAuth

func Auth() IAuth {
	if localAuth == nil {
		panic("implement not found for interface IAuth, forgot register?")
	}
	return localAuth
}

func RegisterAuth(i IAuth) {
	localAuth = i
}
