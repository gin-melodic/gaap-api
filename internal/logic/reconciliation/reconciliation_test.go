package reconciliation

import (
	"testing"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRunEnforcesReadOnlyDatabaseTransaction(t *testing.T) {
	mock, _ := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	testutil.MockMeta(mock, "accounts", []string{
		"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos", "deleted_at",
	})
	testutil.MockMeta(mock, "transactions", []string{
		"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type", "deleted_at",
	})

	mock.ExpectBegin()
	mock.ExpectExec(`SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT .* FROM "?accounts"?`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos", "deleted_at",
		}))
	mock.ExpectQuery(`SELECT .* FROM "?transactions"?`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type", "deleted_at",
		}))
	mock.ExpectCommit()

	report, err := Run(t.Context())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !report.Passed {
		t.Fatalf("empty ledger should reconcile: %+v", report)
	}
	if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
		t.Fatalf("read-only transaction expectations failed: %v", expectationErr)
	}
}

func TestReconcileBalancedLedgerAcrossAccountTypes(t *testing.T) {
	userID := uuid.New()
	assetID := uuid.New()
	liabilityID := uuid.New()
	incomeID := uuid.New()
	expenseID := uuid.New()
	equityID := uuid.New()

	accounts := []entity.Accounts{
		{Id: assetID, UserId: userID, Name: "Cash", Type: utils.AccountTypeAsset, CurrencyCode: "CNY", BalanceUnits: 125, BalanceNanos: 500_000_000},
		{Id: liabilityID, UserId: userID, Name: "Card", Type: utils.AccountTypeLiability, CurrencyCode: "CNY", BalanceUnits: 20},
		{Id: incomeID, UserId: userID, Name: "Salary", Type: utils.AccountTypeIncome, CurrencyCode: "CNY", BalanceUnits: -50},
		{Id: expenseID, UserId: userID, Name: "Food", Type: utils.AccountTypeExpense, CurrencyCode: "CNY", BalanceUnits: 4, BalanceNanos: 500_000_000},
		{Id: equityID, UserId: userID, Name: "Opening", Type: utils.AccountTypeEquity, CurrencyCode: "CNY", BalanceUnits: -100},
	}
	transactions := []entity.Transactions{
		{Id: uuid.New(), UserId: userID, FromAccountId: equityID, ToAccountId: assetID, CurrencyCode: "CNY", BalanceUnits: 100, Type: utils.TransactionTypeOpeningBalance},
		{Id: uuid.New(), UserId: userID, FromAccountId: incomeID, ToAccountId: assetID, CurrencyCode: "CNY", BalanceUnits: 50, Type: utils.TransactionTypeIncome},
		{Id: uuid.New(), UserId: userID, FromAccountId: assetID, ToAccountId: expenseID, CurrencyCode: "CNY", BalanceUnits: 4, BalanceNanos: 500_000_000, Type: utils.TransactionTypeExpense},
		{Id: uuid.New(), UserId: userID, FromAccountId: assetID, ToAccountId: liabilityID, CurrencyCode: "CNY", BalanceUnits: 20, Type: utils.TransactionTypeTransfer},
	}

	report := Reconcile(accounts, transactions)
	if !report.Passed {
		t.Fatalf("expected reconciliation to pass, got differences=%v issues=%v", report.Differences, report.Issues)
	}
}

func TestReconcileDetectsOneNanoDifference(t *testing.T) {
	userID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()
	accounts := []entity.Accounts{
		{Id: fromID, UserId: userID, CurrencyCode: "CNY", BalanceUnits: -1},
		{Id: toID, UserId: userID, CurrencyCode: "CNY", BalanceNanos: 999_999_999},
	}
	transactions := []entity.Transactions{
		{Id: uuid.New(), UserId: userID, FromAccountId: fromID, ToAccountId: toID, CurrencyCode: "CNY", BalanceUnits: 1, Type: utils.TransactionTypeTransfer},
	}

	report := Reconcile(accounts, transactions)
	if report.Passed {
		t.Fatal("expected one-nano difference to fail reconciliation")
	}
	if len(report.Differences) != 1 || report.Differences[0].Difference != "-0.000000001" {
		t.Fatalf("unexpected differences: %+v", report.Differences)
	}
}

func TestReconcileAcceptsNegativeOpeningBalance(t *testing.T) {
	userID := uuid.New()
	equityID := uuid.New()
	liabilityID := uuid.New()
	accounts := []entity.Accounts{
		{Id: equityID, UserId: userID, CurrencyCode: "CNY", BalanceUnits: 25},
		{Id: liabilityID, UserId: userID, CurrencyCode: "CNY", BalanceUnits: -25},
	}
	transactions := []entity.Transactions{
		{Id: uuid.New(), UserId: userID, FromAccountId: equityID, ToAccountId: liabilityID, CurrencyCode: "CNY", BalanceUnits: -25, Type: utils.TransactionTypeOpeningBalance},
	}

	report := Reconcile(accounts, transactions)
	if !report.Passed {
		t.Fatalf("negative opening balance should reconcile: %+v", report)
	}
}

func TestReconcileRejectsCrossUserTransaction(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()
	accounts := []entity.Accounts{
		{Id: fromID, UserId: userID, CurrencyCode: "CNY"},
		{Id: toID, UserId: otherUserID, CurrencyCode: "CNY"},
	}
	transactions := []entity.Transactions{
		{Id: uuid.New(), UserId: userID, FromAccountId: fromID, ToAccountId: toID, CurrencyCode: "CNY", BalanceUnits: 1, Type: utils.TransactionTypeTransfer},
	}

	report := Reconcile(accounts, transactions)
	if report.Passed || len(report.Issues) != 1 {
		t.Fatalf("expected an integrity issue, got %+v", report)
	}
}

func TestReconcileRejectsCurrencyMismatch(t *testing.T) {
	userID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()
	accounts := []entity.Accounts{
		{Id: fromID, UserId: userID, CurrencyCode: "CNY"},
		{Id: toID, UserId: userID, CurrencyCode: "USD"},
	}
	transactions := []entity.Transactions{
		{Id: uuid.New(), UserId: userID, FromAccountId: fromID, ToAccountId: toID, CurrencyCode: "CNY", BalanceUnits: 1, Type: utils.TransactionTypeTransfer},
	}

	report := Reconcile(accounts, transactions)
	if report.Passed || len(report.Issues) != 1 {
		t.Fatalf("expected a currency issue, got %+v", report)
	}
}
