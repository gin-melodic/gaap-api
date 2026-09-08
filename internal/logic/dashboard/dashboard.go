package dashboard

import (
	"context"
	"errors"
	"sort"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	baseCurrency, err := loadUserBaseCurrency(ctx, userId)
	if err != nil {
		return nil, err
	}

	assetBuckets, err := s.loadAccountBuckets(ctx, userId, utils.AccountTypeAsset)
	if err != nil {
		return nil, err
	}
	liabilityBuckets, err := s.loadAccountBuckets(ctx, userId, utils.AccountTypeLiability)
	if err != nil {
		return nil, err
	}

	totalAssets, missingAssets, err := sumValuation(ctx, assetBuckets, baseCurrency)
	if err != nil {
		return nil, err
	}
	totalLiabilities, missingLiabilities, err := sumValuation(ctx, liabilityBuckets, baseCurrency)
	if err != nil {
		return nil, err
	}

	out := &model.DashboardSummary{CurrencyCode: baseCurrency}
	out.AssetsUnits, out.AssetsNanos = decimalToUnitsNanos(totalAssets)
	out.LiabilitiesUnits, out.LiabilitiesNanos = decimalToUnitsNanos(totalLiabilities)
	out.NetWorthUnits, out.NetWorthNanos = decimalToUnitsNanos(totalAssets.Sub(totalLiabilities))
	out.MissingCurrencies = mergeMissing(missingAssets, missingLiabilities)
	return out, nil
}

// GetMonthlyStats returns the monthly income/expense from a Redis snapshot.
func (s *sDashboard) GetMonthlyStats(ctx context.Context) (out *model.MonthlyStats, err error) {
	userId := utils.RequireUserId(ctx)
	return GetMonthlySnapshot(ctx, userId)
}

// loadMonthlyStatsFromDB fetches monthly stats directly from the database.
func (s *sDashboard) loadMonthlyStatsFromDB(ctx context.Context, userId string) (*model.MonthlyStats, error) {
	baseCurrency, err := loadUserBaseCurrency(ctx, userId)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	incomeBuckets, err := s.loadTransactionBuckets(ctx, userId, utils.TransactionTypeIncome, startOfMonth, endOfMonth)
	if err != nil {
		return nil, err
	}
	expenseBuckets, err := s.loadTransactionBuckets(ctx, userId, utils.TransactionTypeExpense, startOfMonth, endOfMonth)
	if err != nil {
		return nil, err
	}

	totalIncome, missingIncome, err := sumValuation(ctx, incomeBuckets, baseCurrency)
	if err != nil {
		return nil, err
	}
	totalExpense, missingExpense, err := sumValuation(ctx, expenseBuckets, baseCurrency)
	if err != nil {
		return nil, err
	}

	out := &model.MonthlyStats{CurrencyCode: baseCurrency}
	out.IncomeUnits, out.IncomeNanos = decimalToUnitsNanos(totalIncome)
	out.ExpenseUnits, out.ExpenseNanos = decimalToUnitsNanos(totalExpense)
	out.MissingCurrencies = mergeMissing(missingIncome, missingExpense)
	return out, nil
}

// loadUserBaseCurrency loads the user's base currency, defaulting to USD when
// it is not configured (legacy users).
func loadUserBaseCurrency(ctx context.Context, userId string) (string, error) {
	var user entity.Users
	err := dao.Users.Ctx(ctx).
		Fields(dao.Users.Columns().MainCurrency).
		Where(dao.Users.Columns().Id, userId).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&user)
	if err != nil {
		return "", gerror.Wrap(err, "failed to load user base currency")
	}
	base := utils.NormalizeCurrency(user.MainCurrency)
	if base == "" {
		return "USD", nil
	}
	return base, nil
}

// loadAccountBuckets sums the balances of a user's accounts of the given type,
// bucketed by currency. Group accounts are excluded.
func (s *sDashboard) loadAccountBuckets(ctx context.Context, userId string, accountType int) (map[string]decimal.Decimal, error) {
	var accounts []entity.Accounts
	err := dao.Accounts.Ctx(ctx).
		Where(dao.Accounts.Columns().UserId, userId).
		Where(dao.Accounts.Columns().Type, accountType).
		Where(dao.Accounts.Columns().IsGroup, false).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		Scan(&accounts)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get accounts")
	}

	buckets := make(map[string]decimal.Decimal)
	for _, account := range accounts {
		money := utils.NewFromEntity(&account)
		currency := utils.NormalizeCurrency(money.Currency)
		if existing, ok := buckets[currency]; ok {
			buckets[currency] = existing.Add(money.Decimal)
		} else {
			buckets[currency] = money.Decimal
		}
	}
	return buckets, nil
}

// loadTransactionBuckets sums transaction amounts of the given type within the
// inclusive date range, bucketed by currency.
func (s *sDashboard) loadTransactionBuckets(ctx context.Context, userId string, transactionType int, start time.Time, end time.Time) (map[string]decimal.Decimal, error) {
	var transactions []entity.Transactions
	err := dao.Transactions.Ctx(ctx).
		Where(dao.Transactions.Columns().UserId, userId).
		Where(dao.Transactions.Columns().Type, transactionType).
		WhereBetween(dao.Transactions.Columns().Date, start, end).
		WhereNull(dao.Transactions.Columns().DeletedAt).
		Scan(&transactions)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get transactions")
	}

	buckets := make(map[string]decimal.Decimal)
	for _, transaction := range transactions {
		money := utils.NewFromTransactions(&transaction)
		currency := utils.NormalizeCurrency(money.Currency)
		if existing, ok := buckets[currency]; ok {
			buckets[currency] = existing.Add(money.Decimal)
		} else {
			buckets[currency] = money.Decimal
		}
	}
	return buckets, nil
}

// sumValuation converts each currency bucket into the base currency and totals
// them. Currencies without a rate are collected and reported as missing.
func sumValuation(ctx context.Context, buckets map[string]decimal.Decimal, base string) (decimal.Decimal, []string, error) {
	total := decimal.NewFromInt(0)
	missing := make([]string, 0)
	for currency, amount := range buckets {
		converted, err := service.ExchangeRate().Convert(ctx, amount, currency, base)
		if err != nil {
			if errors.Is(err, model.ErrMissingRate) {
				missing = append(missing, currency)
				continue
			}
			return decimal.Zero, nil, err
		}
		total = total.Add(converted)
	}
	sort.Strings(missing)
	return total, missing, nil
}

// decimalToUnitsNanos converts an exact decimal (rounded to 9 places) into
// units/nanos components compatible with the Money representation.
func decimalToUnitsNanos(value decimal.Decimal) (int64, int32) {
	money := &utils.MoneyHelper{Decimal: value.Round(9)}
	units, nanos := money.ToEntityValues()
	return units, nanos
}

// mergeMissing returns a sorted, de-duplicated list of missing currencies.
func mergeMissing(groups ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		for _, currency := range group {
			if _, ok := seen[currency]; ok {
				continue
			}
			seen[currency] = struct{}{}
			out = append(out, currency)
		}
	}
	sort.Strings(out)
	return out
}

// GetBalanceTrend returns inclusive daily balance snapshots for the requested
// range. Omitting both dates defaults to the past 60 calendar days.
func (s *sDashboard) GetBalanceTrend(
	ctx context.Context,
	accounts []uuid.UUID,
	startDateValue string,
	endDateValue string,
) (out []model.DailyBalance, err error) {
	now := time.Now()
	isDefaultRange := startDateValue == "" && endDateValue == ""
	startDate, endDate, err := resolveTrendDateRange(now, startDateValue, endDateValue)
	if err != nil {
		return nil, err
	}

	userId := utils.RequireUserId(ctx)
	earliestAccountDate, err := loadEarliestTrendAccountDate(ctx, userId, now.Location())
	if err != nil {
		return nil, err
	}
	startDate, err = enforceEarliestTrendAccountDate(startDate, earliestAccountDate, isDefaultRange)
	if err != nil {
		return nil, err
	}
	return GetTrendSnapshot(ctx, userId, accounts, startDate, endDate)
}

func enforceEarliestTrendAccountDate(startDate time.Time, earliestAccountDate *time.Time, isDefaultRange bool) (time.Time, error) {
	if earliestAccountDate == nil || !startDate.Before(*earliestAccountDate) {
		return startDate, nil
	}
	if !isDefaultRange {
		return time.Time{}, gerror.New("start date must not be before the first account date")
	}
	return *earliestAccountDate, nil
}

func loadEarliestTrendAccountDate(ctx context.Context, userId string, location *time.Location) (*time.Time, error) {
	var accounts []entity.Accounts
	err := dao.Accounts.Ctx(ctx).
		Where(dao.Accounts.Columns().UserId, userId).
		Where(dao.Accounts.Columns().IsGroup, false).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		Fields(dao.Accounts.Columns().Date, dao.Accounts.Columns().CreatedAt).
		Scan(&accounts)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to load first account date")
	}

	return earliestTrendAccountDate(accounts, location), nil
}

func earliestTrendAccountDate(accounts []entity.Accounts, location *time.Location) *time.Time {
	var earliest *time.Time
	for _, account := range accounts {
		var candidate time.Time
		if account.Date != nil {
			candidate = account.Date.Time
		} else if account.CreatedAt != nil {
			candidate = account.CreatedAt.Time
		} else {
			continue
		}
		candidate = candidate.In(location)
		candidate = time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 0, 0, 0, 0, location)
		if earliest == nil || candidate.Before(*earliest) {
			value := candidate
			earliest = &value
		}
	}
	return earliest
}

func resolveTrendDateRange(now time.Time, startDateValue string, endDateValue string) (time.Time, time.Time, error) {
	today := startOfLocalDay(now)
	earliestDate := today.AddDate(-2, 0, 0)

	if startDateValue == "" && endDateValue == "" {
		return today.AddDate(0, 0, -59), today, nil
	}
	if startDateValue == "" || endDateValue == "" {
		return time.Time{}, time.Time{}, gerror.New("start date and end date must be provided together")
	}

	startDate, err := time.ParseInLocation("2006-01-02", startDateValue, now.Location())
	if err != nil {
		return time.Time{}, time.Time{}, gerror.New("invalid start date format (expected YYYY-MM-DD)")
	}
	endDate, err := time.ParseInLocation("2006-01-02", endDateValue, now.Location())
	if err != nil {
		return time.Time{}, time.Time{}, gerror.New("invalid end date format (expected YYYY-MM-DD)")
	}

	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, gerror.New("end date must not be before start date")
	}
	if endDate.After(today) {
		return time.Time{}, time.Time{}, gerror.New("end date must not be in the future")
	}
	if startDate.Before(earliestDate) {
		return time.Time{}, time.Time{}, gerror.New("start date must be within the past two years")
	}
	if startDate.Before(endDate.AddDate(-2, 0, 0)) {
		return time.Time{}, time.Time{}, gerror.New("date range cannot exceed two years")
	}

	return startDate, endDate, nil
}
