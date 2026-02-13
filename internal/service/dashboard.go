// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"

	"github.com/google/uuid"
)

type (
	IDashboard interface {
		// GetDashboardSummary calculates total assets, liabilities, and net worth for the current user
		GetDashboardSummary(ctx context.Context) (out *model.DashboardSummary, err error)
		// GetMonthlyStats calculates income and expense for the current month
		GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error)
		// GetBalanceTrend returns daily balance snapshots for specified accounts
		GetBalanceTrend(ctx context.Context, accounts []uuid.UUID) (out []model.DailyBalance, err error)
	}
)

var (
	localDashboard IDashboard
)

func Dashboard() IDashboard {
	if localDashboard == nil {
		panic("implement not found for interface IDashboard, forgot register?")
	}
	return localDashboard
}

func RegisterDashboard(i IDashboard) {
	localDashboard = i
}
