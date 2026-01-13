package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfListAccounts(ctx context.Context, req *v1.GfListAccountsReq) (res *v1.GfListAccountsRes, err error) {
	queryInput := protoQueryToInput(req.GetQuery())

	accounts, total, err := service.Account().ListAccounts(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int32(0)
	if queryInput.Limit > 0 {
		totalPages = int32((total + queryInput.Limit - 1) / queryInput.Limit)
	}

	return &v1.ListAccountsRes{
		Data: entitiesToProtos(accounts),
		Pagination: &base.PaginatedResponse{
			Total:      int32(total),
			Page:       int32(queryInput.Page),
			Limit:      int32(queryInput.Limit),
			TotalPages: totalPages,
		},
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
