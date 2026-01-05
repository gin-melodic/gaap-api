package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
)

type DashboardSummary struct {
	Assets      float64 `json:"assets"`
	Liabilities float64 `json:"liabilities"`
	NetWorth    float64 `json:"netWorth"`
}

type MonthlyStats struct {
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type DailyBalance struct {
	Date     string             `json:"date"`
	Balances map[string]float64 `json:"balances"`
}

type GetDashboardSummaryReq struct {
	g.Meta `path:"/v1/dashboard/summary" tags:"Dashboard" method:"get" summary:"Get dashboard summary"`
}

type GetDashboardSummaryRes struct {
	g.Meta `mime:"application/json"`
	*DashboardSummary
	*common.BaseResponse
}

type GetMonthlyStatsReq struct {
	g.Meta `path:"/v1/dashboard/monthly-stats" tags:"Dashboard" method:"get" summary:"Get monthly income and expense statistics"`
}

type GetMonthlyStatsRes struct {
	g.Meta `mime:"application/json"`
	*MonthlyStats
	*common.BaseResponse
}

type GetBalanceTrendReq struct {
	g.Meta   `path:"/v1/dashboard/balance-trend" tags:"Dashboard" method:"get" summary:"Get balance trend data for the last 30 days"`
	Accounts []string `json:"accounts" v:"max-length:50"`
}

type GetBalanceTrendRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	Data []DailyBalance `json:"data"`
}
