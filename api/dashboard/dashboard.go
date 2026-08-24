// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package dashboard

import (
	"context"

	"gaap-api/api/dashboard/v1"
)

type IDashboardV1 interface {
	GfGetDashboardSummary(ctx context.Context, req *v1.GfGetDashboardSummaryReq) (res *v1.GfGetDashboardSummaryRes, err error)
	GfGetMonthlyStats(ctx context.Context, req *v1.GfGetMonthlyStatsReq) (res *v1.GfGetMonthlyStatsRes, err error)
	GfGetBalanceTrend(ctx context.Context, req *v1.GfGetBalanceTrendReq) (res *v1.GfGetBalanceTrendRes, err error)
}
