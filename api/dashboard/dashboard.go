package dashboard

import (
	"context"
	v1 "gaap-api/api/dashboard/v1"
)

type IDashboardV1 interface {
	GetDashboardSummary(ctx context.Context, req *v1.GetDashboardSummaryReq) (res *v1.GetDashboardSummaryRes, err error)
	GetMonthlyStats(ctx context.Context, req *v1.GetMonthlyStatsReq) (res *v1.GetMonthlyStatsRes, err error)
	GetBalanceTrend(ctx context.Context, req *v1.GetBalanceTrendReq) (res *v1.GetBalanceTrendRes, err error)
}
