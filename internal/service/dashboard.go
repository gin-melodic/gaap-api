package service

import (
	"context"
	"gaap-api/internal/model"
)

type IDashboard interface {
	GetDashboardSummary(ctx context.Context) (out *model.DashboardSummary, err error)
	GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error)
	GetBalanceTrend(ctx context.Context, accounts []string) (out []model.DailyBalance, err error)
}

var localDashboard IDashboard

func Dashboard() IDashboard {
	if localDashboard == nil {
		panic("implement not found for interface IDashboard, forgot register?")
	}
	return localDashboard
}

func RegisterDashboard(i IDashboard) {
	localDashboard = i
}
