package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/internal/service"
	"gaap-api/utility/proto"
)

func (c *ControllerV1) GfUpdatePassword(ctx context.Context, req *v1.GfUpdatePasswordReq) (res *v1.GfUpdatePasswordRes, err error) {
	if err := proto.ParseFromALE(ctx, &req.UpdatePasswordReq); err != nil {
		return nil, err
	}

	err = service.Auth().UpdatePassword(ctx, req.Password, req.NewPassword, req.ConfirmPassword)
	if err != nil {
		return nil, err
	}
	return &v1.GfUpdatePasswordRes{}, nil
}
