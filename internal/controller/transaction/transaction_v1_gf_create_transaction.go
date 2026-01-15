package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfCreateTransaction(ctx context.Context, req *v1.GfCreateTransactionReq) (res *v1.GfCreateTransactionRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.CreateTransactionReq); err != nil {
		return nil, err
	}

	input := protoInputToCreateInput(ctx, req.GetInput())

	tx, err := service.Transaction().CreateTransaction(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.CreateTransactionRes{
		Transaction: entityToProto(tx),
		Base:        &base.BaseResponse{Message: "success"},
	}, nil
}
