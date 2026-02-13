package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/gogf/gf/v2/net/ghttp"
)

func (c *ControllerV1) GfLogout(ctx context.Context, req *v1.GfLogoutReq) (res *v1.GfLogoutRes, err error) {
	// Parse protobuf from ALE context (LogoutReq has no fields, but we call for consistency)
	if err := utilproto.ParseFromALE(ctx, &req.LogoutReq); err != nil {
		return nil, err
	}

	// Get the token from context and add it to blacklist
	tokenString := ghttp.RequestFromCtx(ctx).GetHeader("Authorization")
	if tokenString != "" {
		service.Auth().AddTokenToBlacklist(ctx, tokenString)
	}

	return &v1.LogoutRes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
