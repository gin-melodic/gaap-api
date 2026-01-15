package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfGetAccountTransactionCount(ctx context.Context, req *v1.GfGetAccountTransactionCountReq) (res *v1.GfGetAccountTransactionCountRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetAccountTransactionCountReq); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	count, err := service.Account().GetAccountTransactionCount(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.GetAccountTransactionCountRes{
		Count: int32(count),
		Base:  &base.BaseResponse{Message: "success"},
	}, nil
}
