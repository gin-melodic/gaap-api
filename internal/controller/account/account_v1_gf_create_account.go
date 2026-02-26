package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/gogf/gf/v2/frame/g"
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

	// 增加出参转换日志
	g.Log().Infof(ctx, "CreateAccount Response: %v", req)

	return &v1.CreateAccountRes{
		Account: entityToProto(account),
		Base:    &base.BaseResponse{Message: "success"},
	}, nil
}
