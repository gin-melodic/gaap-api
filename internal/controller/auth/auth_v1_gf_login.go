package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfLogin(ctx context.Context, req *v1.GfLoginReq) (res *v1.GfLoginRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.LoginReq); err != nil {
		return nil, err
	}

	input := model.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
		Code:     req.GetCode(),
	}

	authResp, err := service.Auth().Login(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.LoginRes{
		Auth: authResponseToProto(authResp),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
