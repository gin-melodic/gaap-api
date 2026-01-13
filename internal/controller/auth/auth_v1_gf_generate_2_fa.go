package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfGenerate2FA(ctx context.Context, req *v1.GfGenerate2FAReq) (res *v1.GfGenerate2FARes, err error) {
	secret, err := service.Auth().Generate2FA(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.Generate2FARes{
		Secret: twoFactorSecretToProto(secret),
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
