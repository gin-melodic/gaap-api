package dashboard

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/dashboard/v1"
	"gaap-api/internal/service"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfGetBalanceTrend(ctx context.Context, req *v1.GfGetBalanceTrendReq) (res *v1.GfGetBalanceTrendRes, err error) {
	// Convert string account IDs to UUIDs
	accountIds := []uuid.UUID{}
	for _, idStr := range req.Accounts {
		if id, err := uuid.Parse(idStr); err == nil {
			accountIds = append(accountIds, id)
		}
	}

	balances, err := service.Dashboard().GetBalanceTrend(ctx, accountIds)
	if err != nil {
		return nil, err
	}

	return &v1.GetBalanceTrendRes{
		Data: dailyBalancesToProtos(balances),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
