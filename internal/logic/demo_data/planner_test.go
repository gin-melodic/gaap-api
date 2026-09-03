package demo_data

import (
	"reflect"
	"testing"
	"time"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPlanTransactionsIsDeterministicAndUsesFractionalAmounts(t *testing.T) {
	location := mustLocation(t)
	user := entity.Users{Id: uuid.MustParse("7851bb7f-0120-4bc5-85a4-52d5953dbe21"), MainCurrency: "USD"}
	accounts := completeAccountSet(user.Id, 20_000)
	date := time.Date(2026, time.April, 10, 0, 0, 0, 0, location)

	first, err := planTransactions(user, accounts, date, location)
	if err != nil {
		t.Fatalf("first plan failed: %v", err)
	}
	second, err := planTransactions(user, accounts, date, location)
	if err != nil {
		t.Fatalf("second plan failed: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("the same user and business date produced different plans")
	}
	if len(first) == 0 || len(first) > 4 {
		t.Fatalf("planned transaction count = %d, want 1..4 on salary day", len(first))
	}
	for _, transaction := range first {
		if transaction.input.BalanceUnits == 0 && transaction.input.BalanceNanos == 0 {
			t.Fatal("planned a zero amount")
		}
		if transaction.input.BalanceNanos == 0 {
			t.Fatal("planned an integer amount")
		}
		if transaction.input.BalanceNanos%10_000_000 != 0 {
			t.Fatalf("amount nanos %d does not represent cents", transaction.input.BalanceNanos)
		}
		parsed, parseErr := time.Parse(time.RFC3339, transaction.input.Date)
		if parseErr != nil {
			t.Fatalf("invalid transaction timestamp: %v", parseErr)
		}
		if parsed.In(location).Format(time.DateOnly) != date.Format(time.DateOnly) {
			t.Fatalf("transaction escaped business date: %s", transaction.input.Date)
		}
	}
}

func TestFitToAvailableBalanceBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		balance     string
		requested   string
		want        string
		wantAllowed bool
	}{
		{name: "exact fractional balance", balance: "10.37", requested: "10.37", want: "10.37", wantAllowed: true},
		{name: "integer cap leaves a cent", balance: "10.00", requested: "20.25", want: "9.99", wantAllowed: true},
		{name: "one cent can be exhausted", balance: "0.01", requested: "20.25", want: "0.01", wantAllowed: true},
		{name: "sub-cent balance is unusable", balance: "0.009", requested: "20.25", wantAllowed: false},
		{name: "zero balance is unusable", balance: "0", requested: "20.25", wantAllowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &accountState{account: entity.Accounts{Type: utils.AccountTypeAsset}, balance: decimal.RequireFromString(test.balance)}
			amount, allowed := fitToAvailableBalance(decimal.RequireFromString(test.requested), state)
			if allowed != test.wantAllowed {
				t.Fatalf("allowed = %v, want %v", allowed, test.wantAllowed)
			}
			if allowed && !amount.Equal(decimal.RequireFromString(test.want)) {
				t.Fatalf("amount = %s, want %s", amount, test.want)
			}
			if allowed && amount.GreaterThan(state.balance) {
				t.Fatalf("amount %s exceeds balance %s", amount, state.balance)
			}
		})
	}
}

func TestPlanRejectsExistingNegativeFinancialAccount(t *testing.T) {
	location := mustLocation(t)
	user := entity.Users{Id: uuid.New(), MainCurrency: "USD"}
	accounts := completeAccountSet(user.Id, 100)
	accounts[0].BalanceUnits = -1

	if _, err := planTransactions(user, accounts, time.Date(2026, time.April, 10, 0, 0, 0, 0, location), location); err == nil {
		t.Fatal("expected a negative asset balance to be rejected")
	}
}

func TestDurationUntilNextMidnightHandlesDST(t *testing.T) {
	location := mustLocation(t)
	fallBackDay := time.Date(2026, time.November, 1, 0, 0, 0, 0, location)
	if got := durationUntilNextMidnight(fallBackDay, location); got != 25*time.Hour {
		t.Fatalf("fall-back day duration = %s, want 25h", got)
	}
	springForwardDay := time.Date(2026, time.March, 8, 0, 0, 0, 0, location)
	if got := durationUntilNextMidnight(springForwardDay, location); got != 23*time.Hour {
		t.Fatalf("spring-forward day duration = %s, want 23h", got)
	}
}

func TestCatchUpEndDateIncludesRequiredInitialBackfill(t *testing.T) {
	location := mustLocation(t)
	config := Config{
		InitialBackfillEndDate: time.Date(2026, time.August, 23, 0, 0, 0, 0, location),
		Location:               location,
	}
	whileLosAngelesIsStillAugust23 := time.Date(2026, time.August, 24, 6, 0, 0, 0, time.UTC)
	if got := catchUpEndDate(whileLosAngelesIsStillAugust23, config); got.Format(time.DateOnly) != "2026-08-23" {
		t.Fatalf("catch-up end = %s, want 2026-08-23", got.Format(time.DateOnly))
	}
	afterInitialBackfill := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	if got := catchUpEndDate(afterInitialBackfill, config); got.Format(time.DateOnly) != "2026-08-25" {
		t.Fatalf("ongoing catch-up end = %s, want 2026-08-25", got.Format(time.DateOnly))
	}
}

func TestRecurringSalaryUsesCalendarDaysAcrossDST(t *testing.T) {
	location := mustLocation(t)
	date := time.Date(2026, time.November, 6, 0, 0, 0, 0, location)
	templates := recurringTemplates(date)
	found := false
	for _, template := range templates {
		if template.note == "Biweekly payroll deposit" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected biweekly salary recurrence after DST transition")
	}
}

func mustLocation(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(defaultTimezone)
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return location
}

func completeAccountSet(userID uuid.UUID, assetUnits int64) []entity.Accounts {
	return []entity.Accounts{
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000001"), UserId: userID, Name: "Chase Total Checking", Type: utils.AccountTypeAsset, CurrencyCode: "USD", BalanceUnits: assetUnits},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000002"), UserId: userID, Name: "Ally Online Savings", Type: utils.AccountTypeAsset, CurrencyCode: "USD", BalanceUnits: assetUnits},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000003"), UserId: userID, Name: "Toyota Auto Loan", Type: utils.AccountTypeLiability, CurrencyCode: "USD", BalanceUnits: 5_000},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000004"), UserId: userID, Name: "Salary Income", Type: utils.AccountTypeIncome, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000005"), UserId: userID, Name: "Freelance Income", Type: utils.AccountTypeIncome, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000006"), UserId: userID, Name: "Interest Income", Type: utils.AccountTypeIncome, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000007"), UserId: userID, Name: "Rent Expense", Type: utils.AccountTypeExpense, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000008"), UserId: userID, Name: "Subscriptions", Type: utils.AccountTypeExpense, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000009"), UserId: userID, Name: "Auto Insurance", Type: utils.AccountTypeExpense, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000010"), UserId: userID, Name: "Dining Out", Type: utils.AccountTypeExpense, CurrencyCode: "USD"},
		{Id: uuid.MustParse("00000000-0000-0000-0000-000000000011"), UserId: userID, Name: "Groceries", Type: utils.AccountTypeExpense, CurrencyCode: "USD"},
	}
}
