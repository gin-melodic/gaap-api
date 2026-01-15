package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfEnable2FA(ctx context.Context, req *v1.GfEnable2FAReq) (res *v1.GfEnable2FARes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.Enable2FAReq); err != nil {
		return nil, err
	}

	err = service.Auth().Enable2FA(ctx, req.GetCode())
	if err != nil {
		return nil, err
	}

	return &v1.Enable2FARes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
