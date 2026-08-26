// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"
)

type (
	IAuth interface {
		Login(ctx context.Context, in model.LoginInput) (out *model.AuthResponse, err error)
		// DemoLogin authenticates the configured demo user without requiring a
		// browser-supplied password or Turnstile token.
		DemoLogin(ctx context.Context) (out *model.AuthResponse, err error)
		Register(ctx context.Context, in model.RegisterInput) (out *model.AuthResponse, err error)
		Generate2FA(ctx context.Context) (out *model.TwoFactorSecret, err error)
		Enable2FA(ctx context.Context, code string) (err error)
		Disable2FA(ctx context.Context, code string, password string) (err error)
		// RefreshToken validates a refresh token and returns a new token pair
		RefreshToken(ctx context.Context, refreshTokenStr string) (out *model.TokenPair, err error)
		// AddTokenToBlacklist adds a token to the blacklist
		AddTokenToBlacklist(ctx context.Context, tokenStr string)
		// IsTokenBlacklisted checks if a token is in the blacklist
		IsTokenBlacklisted(ctx context.Context, token string) bool
		UpdatePassword(ctx context.Context, password string, newPassword string, confirmPassword string) error
		// GetCurrencyList returns a list of all supported currencies
		GetCurrencyList(ctx context.Context) ([]string, error)
	}
)

var (
	localAuth IAuth
)

func Auth() IAuth {
	if localAuth == nil {
		panic("implement not found for interface IAuth, forgot register?")
	}
	return localAuth
}

func RegisterAuth(i IAuth) {
	localAuth = i
}
