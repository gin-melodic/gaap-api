package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfUpdateTransaction(ctx context.Context, req *v1.GfUpdateTransactionReq) (res *v1.GfUpdateTransactionRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.UpdateTransactionReq); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	input := protoInputToUpdateInput(req.GetInput())

	tx, err := service.Transaction().UpdateTransaction(ctx, id, input)
	if err != nil {
		return nil, err
	}

	return &v1.GfUpdateTransactionRes{
		Transaction: entityToProto(tx),
		Base:        &base.BaseResponse{Message: "success"},
	}, nil
}
