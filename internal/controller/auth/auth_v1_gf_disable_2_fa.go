package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfDisable2FA(ctx context.Context, req *v1.GfDisable2FAReq) (res *v1.GfDisable2FARes, err error) {
	err = service.Auth().Disable2FA(ctx, req.GetCode(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &v1.Disable2FARes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
