package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfCreateTransaction(ctx context.Context, req *v1.GfCreateTransactionReq) (res *v1.GfCreateTransactionRes, err error) {
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
