package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfDisable2FA(ctx context.Context, req *v1.GfDisable2FAReq) (res *v1.GfDisable2FARes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.Disable2FAReq); err != nil {
		return nil, err
	}

	err = service.Auth().Disable2FA(ctx, req.GetCode(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &v1.Disable2FARes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
