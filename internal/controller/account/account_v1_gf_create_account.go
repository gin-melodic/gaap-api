package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfCreateAccount(ctx context.Context, req *v1.GfCreateAccountReq) (res *v1.GfCreateAccountRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.CreateAccountReq); err != nil {
		return nil, err
	}

	input := protoInputToCreateInput(ctx, req.GetInput())

	account, err := service.Account().CreateAccount(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.CreateAccountRes{
		Account: entityToProto(account),
		Base:    &base.BaseResponse{Message: "success"},
	}, nil
}
