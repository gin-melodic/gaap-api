package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfGenerate2FA(ctx context.Context, req *v1.GfGenerate2FAReq) (res *v1.GfGenerate2FARes, err error) {
	// Parse protobuf from ALE context (Generate2FAReq has no fields, but we call for consistency)
	if err := utilproto.ParseFromALE(ctx, &req.Generate2FAReq); err != nil {
		return nil, err
	}

	secret, err := service.Auth().Generate2FA(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.Generate2FARes{
		Secret: twoFactorSecretToProto(secret),
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
