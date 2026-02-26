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
		// GetDashboardSummary returns the dashboard summary from a Redis snapshot.
		// The snapshot is pre-computed asynchronously via RabbitMQ whenever transactions
		// or account balances change. Falls back to DB computation on cold start / cache miss.
		GetDashboardSummary(ctx context.Context) (out *model.DashboardSummary, err error)
		// GetMonthlyStats returns the monthly income/expense from a Redis snapshot.
		GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error)
		// GetBalanceTrend returns daily balance snapshots from Redis.
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
