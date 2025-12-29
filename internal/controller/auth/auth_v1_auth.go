package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	common "gaap-api/api/common/v1"
	userV1 "gaap-api/api/user/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/net/ghttp"
)

func (c *ControllerV1) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	in := model.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		Code:     req.Code,
	}
	out, err := service.Auth().Login(ctx, in)
	if err != nil {
		return nil, err
	}
	res = &v1.LoginRes{
		AuthResponse: &v1.AuthResponse{
			Token:        out.AccessToken, // Deprecated, for backward compatibility
			AccessToken:  out.AccessToken,
			RefreshToken: out.RefreshToken,
			User: &userV1.User{
				Email:            out.User.Email,
				Nickname:         out.User.Nickname,
				Avatar:           &out.User.Avatar,
				Plan:             out.User.Plan,
				TwoFactorEnabled: out.User.TwoFactorEnabled,
			},
		},
	}
	return
}

func (c *ControllerV1) Register(ctx context.Context, req *v1.RegisterReq) (res *v1.RegisterRes, err error) {
	in := model.RegisterInput{
		Email:               req.Email,
		Password:            req.Password,
		Nickname:            req.Nickname,
		CfTurnstileResponse: req.CfTurnstileResponse,
	}
	out, err := service.Auth().Register(ctx, in)
	if err != nil {
		return &v1.RegisterRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.RegisterRes{
		AuthResponse: &v1.AuthResponse{
			Token:        out.AccessToken, // Deprecated, for backward compatibility
			AccessToken:  out.AccessToken,
			RefreshToken: out.RefreshToken,
			User: &userV1.User{
				Email:            out.User.Email,
				Nickname:         out.User.Nickname,
				Avatar:           &out.User.Avatar,
				Plan:             out.User.Plan,
				TwoFactorEnabled: out.User.TwoFactorEnabled,
			},
		},
	}
	return
}

func (c *ControllerV1) Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error) {
	// Add token to blacklist
	token := ghttp.RequestFromCtx(ctx).GetHeader("Authorization")
	if token != "" {
		service.Auth().AddTokenToBlacklist(ctx, token)
	}
	res = &v1.LogoutRes{}
	return
}

func (c *ControllerV1) RefreshToken(ctx context.Context, req *v1.RefreshTokenReq) (res *v1.RefreshTokenRes, err error) {
	out, err := service.Auth().RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return &v1.RefreshTokenRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.RefreshTokenRes{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	}
	return
}

func (c *ControllerV1) Generate2FA(ctx context.Context, req *v1.Generate2FAReq) (res *v1.Generate2FARes, err error) {
	out, err := service.Auth().Generate2FA(ctx)
	if err != nil {
		return &v1.Generate2FARes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.Generate2FARes{
		TwoFactorSecret: &v1.TwoFactorSecret{
			Secret: out.Secret,
			Url:    out.Url,
		},
	}
	return
}

func (c *ControllerV1) Enable2FA(ctx context.Context, req *v1.Enable2FAReq) (res *v1.Enable2FARes, err error) {
	err = service.Auth().Enable2FA(ctx, req.Code)
	if err != nil {
		return &v1.Enable2FARes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.Enable2FARes{}
	return
}

func (c *ControllerV1) Disable2FA(ctx context.Context, req *v1.Disable2FAReq) (res *v1.Disable2FARes, err error) {
	err = service.Auth().Disable2FA(ctx, req.Code, req.Password)
	if err != nil {
		return &v1.Disable2FARes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.Disable2FARes{}
	return
}
