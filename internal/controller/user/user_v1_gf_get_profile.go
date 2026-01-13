package user

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/user/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfGetProfile(ctx context.Context, req *v1.GfGetProfileReq) (res *v1.GfGetProfileRes, err error) {
	profile, err := service.User().GetUserProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GfGetProfileRes{
		User: userProfileToProto(profile),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
