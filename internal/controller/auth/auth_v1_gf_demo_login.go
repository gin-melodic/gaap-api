package auth

import (
	"context"

	"gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfDemoLogin(ctx context.Context, req *v1.GfDemoLoginReq) (res *v1.GfDemoLoginRes, err error) {
	if err := utilproto.ParseFromALE(ctx, &req.DemoLoginReq); err != nil {
		return nil, err
	}
	authResp, err := service.Auth().DemoLogin(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.LoginRes{
		Auth: authResponseToProto(authResp),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
