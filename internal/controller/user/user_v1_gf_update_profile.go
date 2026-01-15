package user

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/user/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfUpdateProfile(ctx context.Context, req *v1.GfUpdateProfileReq) (res *v1.GfUpdateProfileRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.UpdateUserProfileReq); err != nil {
		return nil, err
	}

	input := model.UserUpdateInput{
		Nickname: req.Input.Nickname,
	}

	if req.Input.Avatar != nil {
		input.Avatar = *req.Input.Avatar
	}

	profile, err := service.User().UpdateUserProfile(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.GfUpdateProfileRes{
		User: userProfileToProto(profile),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
