package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfRefreshToken(ctx context.Context, req *v1.GfRefreshTokenReq) (res *v1.GfRefreshTokenRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.RefreshTokenReq); err != nil {
		return nil, err
	}

	tokenPair, err := service.Auth().RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, err
	}

	return &v1.RefreshTokenRes{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		SessionKey:   tokenPair.SessionKey,
		Base:         &base.BaseResponse{Message: "success"},
	}, nil
}
