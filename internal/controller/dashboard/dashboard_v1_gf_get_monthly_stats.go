package dashboard

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/dashboard/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfGetMonthlyStats(ctx context.Context, req *v1.GfGetMonthlyStatsReq) (res *v1.GfGetMonthlyStatsRes, err error) {
	stats, err := service.Dashboard().GetMonthlyStats(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GetMonthlyStatsRes{
		Stats: monthlyStatsToProto(stats),
		Base:  &base.BaseResponse{Message: "success"},
	}, nil
}
