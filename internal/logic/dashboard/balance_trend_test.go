package dashboard

import (
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
		trend := calculateBalanceTrend(now, map[uuid.UUID]accountBalance{
			assetID: {Id: assetID, BalanceUnits: -10, CurrencyCode: "CNY"},
		}, []entity.Transactions{transaction})

		assertDailyAccountBalance(t, trend, "2026-08-09", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-10", assetID, -10, 0)
		assertDailyAccountBalance(t, trend, "2026-08-13", assetID, -10, 0)
	})

	t.Run("historical update moves the change to the new date", func(t *testing.T) {
		transaction := historicalExpense(userID, assetID, expenseID, "2026-08-12")
		trend := calculateBalanceTrend(now, map[uuid.UUID]accountBalance{
			assetID: {Id: assetID, BalanceUnits: -10, CurrencyCode: "CNY"},
		}, []entity.Transactions{transaction})

		assertDailyAccountBalance(t, trend, "2026-08-10", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-11", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-12", assetID, -10, 0)
	})

	t.Run("historical delete leaves no trend change", func(t *testing.T) {
		trend := calculateBalanceTrend(now, map[uuid.UUID]accountBalance{
			assetID: {Id: assetID, CurrencyCode: "CNY"},
		}, nil)

		assertDailyAccountBalance(t, trend, "2026-08-10", assetID, 0, 0)
		assertDailyAccountBalance(t, trend, "2026-08-13", assetID, 0, 0)
	})
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
