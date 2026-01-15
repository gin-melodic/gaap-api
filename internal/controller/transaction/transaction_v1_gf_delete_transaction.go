package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfDeleteTransaction(ctx context.Context, req *v1.GfDeleteTransactionReq) (res *v1.GfDeleteTransactionRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.DeleteTransactionReq); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	err = service.Transaction().DeleteTransaction(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.DeleteTransactionRes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
