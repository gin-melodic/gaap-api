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

// GetDashboardSummary calculates total assets, liabilities, and net worth for the current user
func (s *sDashboard) GetDashboardSummary(ctx context.Context) (out *model.DashboardSummary, err error) {
	userId := utils.RequireUserId(ctx)

	out = &model.DashboardSummary{}

	// Get all ASSET type accounts and sum their balances using MoneyHelper
	var assetAccounts []entity.Accounts
	err = dao.Accounts.Ctx(ctx).
		Where("user_id", userId).
		Where("type", utils.AccountTypeAsset).
		Where("is_group", false).
		WhereNull("deleted_at").
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
		Where("user_id", userId).
		Where("type", utils.AccountTypeLiability).
		Where("is_group", false).
		WhereNull("deleted_at").
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

// GetMonthlyStats calculates income and expense for the current month
func (s *sDashboard) GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error) {
	userId := utils.RequireUserId(ctx)

	out = &model.MonthlyStats{}

	// Get start and end of current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Get all INCOME transactions this month
	var incomeTransactions []entity.Transactions
	err = dao.Transactions.Ctx(ctx).
		Where("user_id", userId).
		Where("type", utils.TransactionTypeIncome).
		WhereBetween("date", startOfMonth, endOfMonth).
		WhereNull("deleted_at").
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
		Where("user_id", userId).
		Where("type", utils.TransactionTypeExpense).
		WhereBetween("date", startOfMonth, endOfMonth).
		WhereNull("deleted_at").
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

// GetBalanceTrend returns daily balance snapshots for specified accounts
func (s *sDashboard) GetBalanceTrend(ctx context.Context, accounts []uuid.UUID) (out []model.DailyBalance, err error) {
	userId := utils.RequireUserId(ctx)

	// Default to last 30 days
	now := time.Now()
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	startDate := endDate.AddDate(0, 0, -29)
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	// If no specific accounts requested, get all user's non-group accounts
	if len(accounts) == 0 {
		var userAccounts []entity.Accounts
		err = dao.Accounts.Ctx(ctx).
			Where("user_id", userId).
			Where("is_group", false).
			WhereNull("deleted_at").
			Fields("id").
			Scan(&userAccounts)
		if err != nil {
			return nil, gerror.Wrap(err, "failed to get user accounts")
		}
		for _, acc := range userAccounts {
			accounts = append(accounts, acc.Id)
		}
	}

	if len(accounts) == 0 {
		return []model.DailyBalance{}, nil
	}

	// 1. Get CURRENT balances for these accounts
	type AccountBalance struct {
		Id           uuid.UUID
		BalanceUnits int64
		BalanceNanos int
		CurrencyCode string
	}
	currentBalances := make(map[uuid.UUID]AccountBalance)
	var accountRecs []entity.Accounts
	err = dao.Accounts.Ctx(ctx).
		WhereIn("id", accounts).
		Where("user_id", userId).
		Scan(&accountRecs)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get account balances")
	}
	for _, acc := range accountRecs {
		currentBalances[acc.Id] = AccountBalance{
			Id:           acc.Id,
			BalanceUnits: acc.BalanceUnits,
			BalanceNanos: acc.BalanceNanos,
			CurrencyCode: acc.CurrencyCode,
		}
	}

	// 2. Get ALL transactions for these accounts from startDate to NOW
	var fromTrans []entity.Transactions
	var toTrans []entity.Transactions

	err = dao.Transactions.Ctx(ctx).
		WhereIn("from_account_id", accounts).
		WhereGTE("date", startOfDay).
		Limit(10000).
		Scan(&fromTrans)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get from transactions")
	}

	err = dao.Transactions.Ctx(ctx).
		WhereIn("to_account_id", accounts).
		WhereGTE("date", startOfDay).
		Limit(10000).
		Scan(&toTrans)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get to transactions")
	}

	// Merge and deduplicate transactions
	txMap := make(map[uuid.UUID]entity.Transactions)
	for _, t := range fromTrans {
		txMap[t.Id] = t
	}
	for _, t := range toTrans {
		txMap[t.Id] = t
	}

	transactions := make([]entity.Transactions, 0, len(txMap))
	for _, t := range txMap {
		transactions = append(transactions, t)
	}

	// Create a map of Date -> Transactions
	transactionsByDate := make(map[string][]entity.Transactions)
	for _, t := range transactions {
		if t.Date == nil {
			continue
		}
		dateStr := t.Date.Layout("2006-01-02")
		transactionsByDate[dateStr] = append(transactionsByDate[dateStr], t)
	}

	// 3. Calculate daily balances BACKWARDS using MoneyHelper
	runningBalances := make(map[uuid.UUID]*utils.MoneyHelper)
	for accId, bal := range currentBalances {
		entity := &entity.Accounts{
			BalanceUnits: bal.BalanceUnits,
			BalanceNanos: bal.BalanceNanos,
			CurrencyCode: bal.CurrencyCode,
		}
		runningBalances[accId] = utils.NewFromEntity(entity)
	}

	dailyMap := make(map[string]map[string]model.DailyAccountBalance)

	cursorDate := endDate
	for !cursorDate.Before(startOfDay) {
		dateStr := cursorDate.Format("2006-01-02")

		// Record the balance at the END of this day
		dayBalances := make(map[string]model.DailyAccountBalance)
		for accId, bal := range runningBalances {
			units, nanos := bal.ToEntityValues()
			dayBalances[accId.String()] = model.DailyAccountBalance{
				Units:        units,
				Nanos:        nanos,
				CurrencyCode: bal.Currency,
			}
		}
		dailyMap[dateStr] = dayBalances

		// Reverse transactions of this day to get start-of-day balances
		if txs, ok := transactionsByDate[dateStr]; ok {
			for _, tx := range txs {
				// Create delta MoneyHelper
				deltaEntity := &entity.Accounts{
					BalanceUnits: tx.BalanceUnits,
					BalanceNanos: tx.BalanceNanos,
					CurrencyCode: tx.CurrencyCode,
				}
				delta := utils.NewFromEntity(deltaEntity)

				// FromAccount: Money left, so add back
				if bal, ok := runningBalances[tx.FromAccountId]; ok {
					newBal, _ := bal.Add(delta)
					if newBal != nil {
						runningBalances[tx.FromAccountId] = newBal
					}
				}
				// ToAccount: Money entered, so subtract
				if bal, ok := runningBalances[tx.ToAccountId]; ok {
					newBal, _ := bal.Sub(delta)
					if newBal != nil {
						runningBalances[tx.ToAccountId] = newBal
					}
				}
			}
		}

		cursorDate = cursorDate.AddDate(0, 0, -1)
	}

	// 4. Construct final output (sorted by date)
	out = make([]model.DailyBalance, 0)
	for d := startOfDay; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if bals, ok := dailyMap[dateStr]; ok {
			out = append(out, model.DailyBalance{
				Date:     dateStr,
				Balances: bals,
			})
		} else {
			out = append(out, model.DailyBalance{
				Date:     dateStr,
				Balances: make(map[string]model.DailyAccountBalance),
			})
		}
	}

	return out, nil
}
