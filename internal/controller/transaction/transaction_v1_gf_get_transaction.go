package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfGetTransaction(ctx context.Context, req *v1.GfGetTransactionReq) (res *v1.GfGetTransactionRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetTransactionReq); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	tx, err := service.Transaction().GetTransaction(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.GetTransactionRes{
		Transaction: entityToProto(tx),
		Base:        &base.BaseResponse{Message: "success"},
	}, nil
}
