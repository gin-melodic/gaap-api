package auth

import (
	"context"

	v1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfRegister(ctx context.Context, req *v1.GfRegisterReq) (res *v1.GfRegisterRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.RegisterReq); err != nil {
		return nil, err
	}

	input := model.RegisterInput{
		Email:               req.GetEmail(),
		Password:            req.GetPassword(),
		Nickname:            req.GetNickname(),
		CfTurnstileResponse: req.GetCfTurnstileResponse(),
	}

	authResp, err := service.Auth().Register(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.RegisterRes{
		Auth: authResponseToProto(authResp),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
