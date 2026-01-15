package transaction

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfListTransactions(ctx context.Context, req *v1.GfListTransactionsReq) (res *v1.GfListTransactionsRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.ListTransactionsReq); err != nil {
		return nil, err
	}

	queryInput := protoQueryToInput(req.GetQuery())

	transactions, total, err := service.Transaction().ListTransactions(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int32(0)
	if queryInput.Limit > 0 {
		totalPages = int32((total + queryInput.Limit - 1) / queryInput.Limit)
	}

	return &v1.ListTransactionsRes{
		Data: entitiesToProtos(transactions),
		Pagination: &base.PaginatedResponse{
			Total:      int32(total),
			Page:       int32(queryInput.Page),
			Limit:      int32(queryInput.Limit),
			TotalPages: totalPages,
		},
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
