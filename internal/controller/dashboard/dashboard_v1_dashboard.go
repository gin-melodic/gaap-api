package dashboard

import (
	"context"
	common "gaap-api/api/common/v1"
	v1 "gaap-api/api/dashboard/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GetDashboardSummary(ctx context.Context, req *v1.GetDashboardSummaryReq) (res *v1.GetDashboardSummaryRes, err error) {
	out, err := service.Dashboard().GetDashboardSummary(ctx)
	if err != nil {
		return &v1.GetDashboardSummaryRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.GetDashboardSummaryRes{
		DashboardSummary: &v1.DashboardSummary{
			Assets:      out.Assets,
			Liabilities: out.Liabilities,
			NetWorth:    out.NetWorth,
		},
	}
	return
}

func (c *ControllerV1) GetMonthlyStats(ctx context.Context, req *v1.GetMonthlyStatsReq) (res *v1.GetMonthlyStatsRes, err error) {
	out, err := service.Dashboard().GetMonthlyStats(ctx)
	if err != nil {
		return &v1.GetMonthlyStatsRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.GetMonthlyStatsRes{
		MonthlyStats: &v1.MonthlyStats{
			Income:  out.Income,
			Expense: out.Expense,
		},
	}
	return
}

func (c *ControllerV1) GetBalanceTrend(ctx context.Context, req *v1.GetBalanceTrendReq) (res *v1.GetBalanceTrendRes, err error) {
	out, err := service.Dashboard().GetBalanceTrend(ctx, req.Accounts)
	if err != nil {
		return &v1.GetBalanceTrendRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	var data []v1.DailyBalance
	for _, d := range out {
		data = append(data, v1.DailyBalance{
			Date:     d.Date,
			Balances: d.Balances,
		})
	}
	res = &v1.GetBalanceTrendRes{
		Data: data,
	}
	return
}
