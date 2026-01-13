package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfUpdateTransaction(ctx context.Context, req *v1.GfUpdateTransactionReq) (res *v1.GfUpdateTransactionRes, err error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	input := protoInputToUpdateInput(req.GetInput())

	tx, err := service.Transaction().UpdateTransaction(ctx, id, input)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateTransactionRes{
		Transaction: entityToProto(tx),
		Base:        &base.BaseResponse{Message: "success"},
	}, nil
}
