package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfGetAccount(ctx context.Context, req *v1.GfGetAccountReq) (res *v1.GfGetAccountRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetAccountReq); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	account, err := service.Account().GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.GetAccountRes{
		Account: entityToProto(account),
		Base:    &base.BaseResponse{Message: "success"},
	}, nil
}
