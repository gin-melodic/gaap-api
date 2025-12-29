package dashboard

import (
	"context"
	"errors"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"
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
	userId := ctx.Value(middleware.UserIdKey)
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	out = &model.DashboardSummary{}

	// Calculate total assets (sum of all ASSET type account balances)
	assetsResult, err := dao.Accounts.Ctx(ctx).
		Where("user_id", userId).
		Where("type", "ASSET").
		Where("is_group", false).
		WhereNull("deleted_at").
		Sum("balance")
	if err != nil {
		return nil, err
	}
	out.Assets = assetsResult

	// Calculate total liabilities (sum of all LIABILITY type account balances)
	liabilitiesResult, err := dao.Accounts.Ctx(ctx).
		Where("user_id", userId).
		Where("type", "LIABILITY").
		Where("is_group", false).
		WhereNull("deleted_at").
		Sum("balance")
	if err != nil {
		return nil, err
	}
	out.Liabilities = liabilitiesResult

	// Calculate net worth
	out.NetWorth = out.Assets - out.Liabilities

	return
}

// GetMonthlyStats calculates income and expense for the current month
func (s *sDashboard) GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error) {
	userId := ctx.Value(middleware.UserIdKey)
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	out = &model.MonthlyStats{}

	// Get start and end of current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Calculate total income this month
	incomeResult, err := dao.Transactions.Ctx(ctx).
		Where("user_id", userId).
		Where("type", "INCOME").
		WhereBetween("date", startOfMonth, endOfMonth).
		WhereNull("deleted_at").
		Sum("amount")
	if err != nil {
		return nil, err
	}
	out.Income = incomeResult

	// Calculate total expense this month
	expenseResult, err := dao.Transactions.Ctx(ctx).
		Where("user_id", userId).
		Where("type", "EXPENSE").
		WhereBetween("date", startOfMonth, endOfMonth).
		WhereNull("deleted_at").
		Sum("amount")
	if err != nil {
		return nil, err
	}
	out.Expense = expenseResult

	return
}

// GetBalanceTrend returns daily balance snapshots for specified accounts
func (s *sDashboard) GetBalanceTrend(ctx context.Context, accounts []string) (out []model.DailyBalance, err error) {
	userId := ctx.Value(middleware.UserIdKey)
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	// Default to last 30 days
	now := time.Now()
	// Set end of today clearly
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	startDate := endDate.AddDate(0, 0, -29) // 30 days including today
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	// If no specific accounts requested, get all user's non-group accounts
	if len(accounts) == 0 {
		res, err := dao.Accounts.Ctx(ctx).
			Where("user_id", userId).
			Where("is_group", false).
			WhereNull("deleted_at").
			Fields("id").
			All()
		if err != nil {
			return nil, err
		}
		accounts = res.Array("id").Strings()
	}

	if len(accounts) == 0 {
		return []model.DailyBalance{}, nil
	}

	// 1. Get CURRENT balances for these accounts
	currentBalances := make(map[string]float64)
	var accountRecs []model.Account
	err = dao.Accounts.Ctx(ctx).
		WhereIn("id", accounts).
		Where("user_id", userId).
		Scan(&accountRecs)
	if err != nil {
		return nil, err
	}
	for _, acc := range accountRecs {
		currentBalances[acc.Id] = acc.Balance
	}

	// 2. Get ALL transactions for these accounts from startDate to NOW
	var transactions []entity.Transactions
	var fromTrans []entity.Transactions
	var toTrans []entity.Transactions

	// Transactions where these accounts are SENDER (money OUT)
	err = dao.Transactions.Ctx(ctx).
		WhereIn("from_account_id", accounts).
		WhereGTE("date", startOfDay).
		Limit(10000).
		Scan(&fromTrans)
	if err != nil {
		return nil, err
	}

	// Transactions where these accounts are RECEIVER (money IN)
	err = dao.Transactions.Ctx(ctx).
		WhereIn("to_account_id", accounts).
		WhereGTE("date", startOfDay).
		Limit(10000).
		Scan(&toTrans)
	if err != nil {
		return nil, err
	}

	// Merge and deduplicate transactions
	txMap := make(map[string]entity.Transactions)
	for _, t := range fromTrans {
		txMap[t.Id] = t
	}
	for _, t := range toTrans {
		txMap[t.Id] = t
	}

	transactions = make([]entity.Transactions, 0, len(txMap))
	for _, t := range txMap {
		transactions = append(transactions, t)
	}

	// Create a map of Date -> Transactions for easier processing
	// We map by YYYY-MM-DD
	transactionsByDate := make(map[string][]entity.Transactions)
	for _, t := range transactions {
		if t.Date == nil {
			continue
		}
		// Use Layout("2006-01-02") to be identical to time.Time.Format
		dateStr := t.Date.Layout("2006-01-02")
		transactionsByDate[dateStr] = append(transactionsByDate[dateStr], t)
	}

	// 3. Calculate daily balances BACKWARDS
	// Initialize running balances with current balances (which corresponds to END of today)
	runningBalances := make(map[string]float64)
	for k, v := range currentBalances {
		runningBalances[k] = v
	}

	// Loop from TODAY backwards to START_DATE
	// We need to output the result in date order (oldest to newest), so we'll store in a temp map or list
	dailyMap := make(map[string]map[string]float64)

	cursorDate := endDate
	for !cursorDate.Before(startOfDay) {
		dateStr := cursorDate.Format("2006-01-02")

		// Record the balance at the END of this day (which is the current runningBalance)
		dayBalances := make(map[string]float64)
		for accId, bal := range runningBalances {
			dayBalances[accId] = bal
		}
		dailyMap[dateStr] = dayBalances

		// Update runningBalances to be the balance at the START of this day (for the next iteration, which is yesterday)
		// To go from End-of-Day to Start-of-Day, we REVERSE the transactions of this day.
		// If money went OUT (FromAccount), we ADD it back.
		// If money went IN (ToAccount), we SUBTRACT it.
		if txs, ok := transactionsByDate[dateStr]; ok {
			for _, tx := range txs {
				// Handle FromAccount (Sender): Money left, so add back
				if _, ok := runningBalances[tx.FromAccountId]; ok {
					runningBalances[tx.FromAccountId] += tx.Amount
				}
				// Handle ToAccount (Receiver): Money entered, so subtract
				if _, ok := runningBalances[tx.ToAccountId]; ok {
					runningBalances[tx.ToAccountId] -= tx.Amount
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
			// Should not happen with above logic, but fallback
			out = append(out, model.DailyBalance{
				Date:     dateStr,
				Balances: make(map[string]float64),
			})
		}
	}

	return
}
