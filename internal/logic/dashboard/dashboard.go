package dashboard

import (
	"context"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/google/uuid"
)

type sDashboard struct{}

func init() {
	service.RegisterDashboard(New())
}

func New() *sDashboard {
	return &sDashboard{}
}

// GetDashboardSummary returns the dashboard summary from a Redis snapshot.
// The snapshot is pre-computed asynchronously via RabbitMQ whenever transactions
// or account balances change. Falls back to DB computation on cold start / cache miss.
func (s *sDashboard) GetDashboardSummary(ctx context.Context) (out *model.DashboardSummary, err error) {
	userId := utils.RequireUserId(ctx)
	return GetSummarySnapshot(ctx, userId)
}

// loadDashboardSummaryFromDB fetches dashboard summary directly from the database.
func (s *sDashboard) loadDashboardSummaryFromDB(ctx context.Context, userId string) (*model.DashboardSummary, error) {
	out := &model.DashboardSummary{}

	// Get all ASSET type accounts and sum their balances using MoneyHelper
	var assetAccounts []entity.Accounts
	err := dao.Accounts.Ctx(ctx).
		Where(dao.Accounts.Columns().UserId, userId).
		Where(dao.Accounts.Columns().Type, utils.AccountTypeAsset).
		Where(dao.Accounts.Columns().IsGroup, false).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		Scan(&assetAccounts)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get asset accounts")
	}

	// Sum assets using MoneyHelper for precision
	var totalAssets *utils.MoneyHelper
	for i, acc := range assetAccounts {
		accBalance := utils.NewFromEntity(&acc)
		if i == 0 {
			totalAssets = accBalance
			out.CurrencyCode = acc.CurrencyCode
		} else {
			totalAssets, err = totalAssets.Add(accBalance)
			if err != nil {
				// Currency mismatch - skip this account or handle differently
				continue
			}
		}
	}
	if totalAssets != nil {
		out.AssetsUnits, out.AssetsNanos = totalAssets.ToEntityValues()
	}

	// Get all LIABILITY type accounts
	var liabilityAccounts []entity.Accounts
	err = dao.Accounts.Ctx(ctx).
		Where(dao.Accounts.Columns().UserId, userId).
		Where(dao.Accounts.Columns().Type, utils.AccountTypeLiability).
		Where(dao.Accounts.Columns().IsGroup, false).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		Scan(&liabilityAccounts)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get liability accounts")
	}

	// Sum liabilities
	var totalLiabilities *utils.MoneyHelper
	for i, acc := range liabilityAccounts {
		accBalance := utils.NewFromEntity(&acc)
		if i == 0 {
			totalLiabilities = accBalance
		} else {
			totalLiabilities, err = totalLiabilities.Add(accBalance)
			if err != nil {
				continue
			}
		}
	}
	if totalLiabilities != nil {
		out.LiabilitiesUnits, out.LiabilitiesNanos = totalLiabilities.ToEntityValues()
	}

	// Calculate net worth (Assets - Liabilities)
	if totalAssets != nil && totalLiabilities != nil {
		netWorth, err := totalAssets.Sub(totalLiabilities)
		if err == nil {
			out.NetWorthUnits, out.NetWorthNanos = netWorth.ToEntityValues()
		}
	} else if totalAssets != nil {
		out.NetWorthUnits, out.NetWorthNanos = totalAssets.ToEntityValues()
	}

	return out, nil
}

// GetMonthlyStats returns the monthly income/expense from a Redis snapshot.
func (s *sDashboard) GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error) {
	userId := utils.RequireUserId(ctx)
	return GetMonthlySnapshot(ctx, userId)
}

// loadMonthlyStatsFromDB fetches monthly stats directly from the database.
func (s *sDashboard) loadMonthlyStatsFromDB(ctx context.Context, userId string) (*model.MonthlyStats, error) {
	out := &model.MonthlyStats{}

	// Get start and end of current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Get all INCOME transactions this month
	var incomeTransactions []entity.Transactions
	err := dao.Transactions.Ctx(ctx).
		Where(dao.Transactions.Columns().UserId, userId).
		Where(dao.Transactions.Columns().Type, utils.TransactionTypeIncome).
		WhereBetween(dao.Transactions.Columns().Date, startOfMonth, endOfMonth).
		WhereNull(dao.Transactions.Columns().DeletedAt).
		Scan(&incomeTransactions)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get income transactions")
	}

	// Sum income using MoneyHelper
	var totalIncome *utils.MoneyHelper
	for i, tx := range incomeTransactions {
		// Create a temporary entity to use MoneyHelper
		txEntity := &entity.Accounts{
			BalanceUnits: tx.BalanceUnits,
			BalanceNanos: tx.BalanceNanos,
			CurrencyCode: tx.CurrencyCode,
		}
		txBalance := utils.NewFromEntity(txEntity)
		if i == 0 {
			totalIncome = txBalance
			out.CurrencyCode = tx.CurrencyCode
		} else {
			totalIncome, err = totalIncome.Add(txBalance)
			if err != nil {
				continue
			}
		}
	}
	if totalIncome != nil {
		out.IncomeUnits, out.IncomeNanos = totalIncome.ToEntityValues()
	}

	// Get all EXPENSE transactions this month
	var expenseTransactions []entity.Transactions
	err = dao.Transactions.Ctx(ctx).
		Where(dao.Transactions.Columns().UserId, userId).
		Where(dao.Transactions.Columns().Type, utils.TransactionTypeExpense).
		WhereBetween(dao.Transactions.Columns().Date, startOfMonth, endOfMonth).
		WhereNull(dao.Transactions.Columns().DeletedAt).
		Scan(&expenseTransactions)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get expense transactions")
	}

	// Sum expenses
	var totalExpense *utils.MoneyHelper
	for i, tx := range expenseTransactions {
		txEntity := &entity.Accounts{
			BalanceUnits: tx.BalanceUnits,
			BalanceNanos: tx.BalanceNanos,
			CurrencyCode: tx.CurrencyCode,
		}
		txBalance := utils.NewFromEntity(txEntity)
		if i == 0 {
			totalExpense = txBalance
		} else {
			totalExpense, err = totalExpense.Add(txBalance)
			if err != nil {
				continue
			}
		}
	}
	if totalExpense != nil {
		out.ExpenseUnits, out.ExpenseNanos = totalExpense.ToEntityValues()
	}

	return out, nil
}

// GetBalanceTrend returns daily balance snapshots from Redis.
func (s *sDashboard) GetBalanceTrend(ctx context.Context, accounts []uuid.UUID) (out []model.DailyBalance, err error) {
	userId := utils.RequireUserId(ctx)
	return GetTrendSnapshot(ctx, userId, accounts)
}
