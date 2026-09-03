package dashboard

import (
	"fmt"
	"testing"
	"time"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

func TestCalculateBalanceTrendHistoricalCreateUpdateDelete(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	assetID := uuid.New()
	expenseID := uuid.New()
	userID := uuid.New()

	t.Run("historical create changes its posting date and following days", func(t *testing.T) {
		transaction := historicalExpense(userID, assetID, expenseID, "2026-08-10")
		trend := calculateBalanceTrend(now.AddDate(0, 0, -29), now, map[uuid.UUID]accountBalance{
			assetID: {Id: assetID, BalanceUnits: -10, CurrencyCode: "CNY"},
		}, []entity.Transactions{transaction})

		assertDailyAccountBalance(t, trend, "2026-08-09", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-10", assetID, -10, 0)
		assertDailyAccountBalance(t, trend, "2026-08-13", assetID, -10, 0)
	})

	t.Run("historical update moves the change to the new date", func(t *testing.T) {
		transaction := historicalExpense(userID, assetID, expenseID, "2026-08-12")
		trend := calculateBalanceTrend(now.AddDate(0, 0, -29), now, map[uuid.UUID]accountBalance{
			assetID: {Id: assetID, BalanceUnits: -10, CurrencyCode: "CNY"},
		}, []entity.Transactions{transaction})

		assertDailyAccountBalance(t, trend, "2026-08-10", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-11", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-12", assetID, -10, 0)
	})

	t.Run("historical delete leaves no trend change", func(t *testing.T) {
		trend := calculateBalanceTrend(now.AddDate(0, 0, -29), now, map[uuid.UUID]accountBalance{
			assetID: {Id: assetID, CurrencyCode: "CNY"},
		}, nil)

		assertDailyAccountBalance(t, trend, "2026-08-10", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-13", assetID, 0, 0)
	})
}

func TestResolveTrendDateRange(t *testing.T) {
	now := time.Date(2026, time.August, 24, 16, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	t.Run("defaults to sixty inclusive days", func(t *testing.T) {
		startDate, endDate, err := resolveTrendDateRange(now, "", "")
		if err != nil {
			t.Fatalf("resolve default range: %v", err)
		}
		if got, want := startDate.Format("2006-01-02"), "2026-06-26"; got != want {
			t.Fatalf("start date = %s, want %s", got, want)
		}
		if got, want := endDate.Format("2006-01-02"), "2026-08-24"; got != want {
			t.Fatalf("end date = %s, want %s", got, want)
		}
	})

	validRanges := []struct {
		name  string
		start string
		end   string
	}{
		{name: "same day", start: "2026-08-24", end: "2026-08-24"},
		{name: "earliest rolling date", start: "2024-08-24", end: "2026-08-24"},
	}
	for _, testCase := range validRanges {
		t.Run(testCase.name, func(t *testing.T) {
			if _, _, err := resolveTrendDateRange(now, testCase.start, testCase.end); err != nil {
				t.Fatalf("resolve valid range: %v", err)
			}
		})
	}

	t.Run("leap day within rolling window", func(t *testing.T) {
		leapNow := time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC)
		if _, _, err := resolveTrendDateRange(leapNow, "2024-02-29", "2024-03-01"); err != nil {
			t.Fatalf("resolve leap-day range: %v", err)
		}
	})

	invalidRanges := []struct {
		name  string
		start string
		end   string
	}{
		{name: "missing end", start: "2026-08-01"},
		{name: "missing start", end: "2026-08-24"},
		{name: "malformed start", start: "2026/08/01", end: "2026-08-24"},
		{name: "malformed end", start: "2026-08-01", end: "24-08-2026"},
		{name: "reversed", start: "2026-08-24", end: "2026-08-01"},
		{name: "future", start: "2026-08-24", end: "2026-08-25"},
		{name: "older than rolling window", start: "2024-08-23", end: "2026-08-23"},
		{name: "more than two years", start: "2023-01-01", end: "2025-01-02"},
	}
	for _, testCase := range invalidRanges {
		t.Run(testCase.name, func(t *testing.T) {
			if _, _, err := resolveTrendDateRange(now, testCase.start, testCase.end); err == nil {
				t.Fatal("expected range validation to fail")
			}
		})
	}
}

func TestEarliestTrendAccountDate(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	accounts := []entity.Accounts{
		{Date: gtime.NewFromStr("2025-06-15"), CreatedAt: gtime.NewFromStr("2025-08-01")},
		{Date: gtime.NewFromStr("2025-02-10"), CreatedAt: gtime.NewFromStr("2025-03-01")},
	}

	earliest := earliestTrendAccountDate(accounts, location)
	if earliest == nil || earliest.Format("2006-01-02") != "2025-02-10" {
		t.Fatalf("earliest account date = %v, want 2025-02-10", earliest)
	}

	t.Run("explicit range before first account is rejected", func(t *testing.T) {
		startDate := time.Date(2025, time.January, 1, 0, 0, 0, 0, location)
		if _, err := enforceEarliestTrendAccountDate(startDate, earliest, false); err == nil {
			t.Fatal("expected explicit range before first account to fail")
		}
	})

	t.Run("default range is clamped to first account", func(t *testing.T) {
		startDate := time.Date(2025, time.January, 1, 0, 0, 0, 0, location)
		resolved, err := enforceEarliestTrendAccountDate(startDate, earliest, true)
		if err != nil {
			t.Fatalf("clamp default range: %v", err)
		}
		if resolved.Format("2006-01-02") != "2025-02-10" {
			t.Fatalf("clamped start = %s, want 2025-02-10", resolved.Format("2006-01-02"))
		}
	})

	t.Run("created timestamp is used when opening date is missing", func(t *testing.T) {
		fallback := earliestTrendAccountDate([]entity.Accounts{{CreatedAt: gtime.NewFromStr("2025-04-03 12:00:00")}}, location)
		if fallback == nil || fallback.Format("2006-01-02") != "2025-04-03" {
			t.Fatalf("fallback account date = %v, want 2025-04-03", fallback)
		}
	})
}

func TestCalculateBalanceTrendIncludesRangeBoundariesAndLargeTransactionSet(t *testing.T) {
	startDate := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	assetID := uuid.New()
	expenseID := uuid.New()
	userID := uuid.New()
	transactions := make([]entity.Transactions, 0, 10002)
	for index := 0; index < 10001; index++ {
		transaction := historicalExpense(userID, assetID, expenseID, "2026-08-01")
		transaction.Id = uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("tx-%d", index)))
		transaction.BalanceNanos = 1
		transaction.BalanceUnits = 0
		transactions = append(transactions, transaction)
	}
	lastTransaction := historicalExpense(userID, assetID, expenseID, "2026-08-02")
	transactions = append(transactions, lastTransaction)

	trend := calculateBalanceTrend(startDate, endDate, map[uuid.UUID]accountBalance{
		assetID: {Id: assetID, BalanceUnits: -10, BalanceNanos: -10001, CurrencyCode: "CNY"},
	}, transactions)

	if len(trend) != 2 {
		t.Fatalf("trend length = %d, want 2", len(trend))
	}
	assertDailyAccountBalance(t, trend, "2026-08-01", assetID, 0, -10001)
	assertDailyAccountBalance(t, trend, "2026-08-02", assetID, -10, -10001)
}

func TestSliceTrendSnapshotFiltersDatesAndAccounts(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	data := []model.DailyBalance{
		{Date: "2026-08-01", Balances: map[string]model.DailyAccountBalance{firstID.String(): {Units: 1}, secondID.String(): {Units: 2}}},
		{Date: "2026-08-02", Balances: map[string]model.DailyAccountBalance{firstID.String(): {Units: 3}, secondID.String(): {Units: 4}}},
		{Date: "2026-08-03", Balances: map[string]model.DailyAccountBalance{firstID.String(): {Units: 5}, secondID.String(): {Units: 6}}},
	}

	result := sliceTrendSnapshot(
		data,
		[]uuid.UUID{secondID},
		time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC),
	)
	if len(result) != 2 {
		t.Fatalf("result length = %d, want 2", len(result))
	}
	if len(result[0].Balances) != 1 || result[0].Balances[secondID.String()].Units != 4 {
		t.Fatalf("unexpected filtered balances: %#v", result[0].Balances)
	}
}

func historicalExpense(userID, assetID, expenseID uuid.UUID, date string) entity.Transactions {
	return entity.Transactions{
		Id:            uuid.New(),
		UserId:        userID,
		Date:          gtime.NewFromStr(date),
		FromAccountId: assetID,
		ToAccountId:   expenseID,
		CurrencyCode:  "CNY",
		BalanceUnits:  10,
		Type:          utils.TransactionTypeExpense,
	}
}

func assertDailyAccountBalance(
	t *testing.T,
	trend []model.DailyBalance,
	date string,
	accountID uuid.UUID,
	wantUnits int64,
	wantNanos int32,
) {
	t.Helper()
	for _, day := range trend {
		if day.Date != date {
			continue
		}
		balance, ok := day.Balances[accountID.String()]
		if !ok {
			t.Fatalf("date %s has no balance for account %s", date, accountID)
		}
		if balance.Units != wantUnits || balance.Nanos != wantNanos {
			t.Fatalf("date %s balance = %d.%09d, want %d.%09d", date, balance.Units, balance.Nanos, wantUnits, wantNanos)
		}
		return
	}
	t.Fatalf("date %s not found in trend", date)
}
