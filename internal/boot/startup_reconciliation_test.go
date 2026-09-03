package boot

import (
	"errors"
	"testing"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func expectReadOnlyLedgerQueries(mock sqlmock.Sqlmock, accounts, transactions *sqlmock.Rows) {
	mock.ExpectBegin()
	mock.ExpectExec(`SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT .* FROM "?accounts"?`).WillReturnRows(accounts)
	mock.ExpectQuery(`SELECT .* FROM "?transactions"?`).WillReturnRows(transactions)
}

func emptyLedgerRows() (*sqlmock.Rows, *sqlmock.Rows) {
	accounts := sqlmock.NewRows([]string{
		"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos", "deleted_at",
	})
	transactions := sqlmock.NewRows([]string{
		"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type", "deleted_at",
	})
	return accounts, transactions
}

// oneNanoLedgerRows describes an opening balance of exactly 1.000000000 while the
// receiving account persisted 0.999999999: a difference of exactly one nano unit.
func oneNanoLedgerRows() (*sqlmock.Rows, *sqlmock.Rows) {
	userID := uuid.New().String()
	equityID := uuid.New().String()
	assetID := uuid.New().String()

	accounts := sqlmock.NewRows([]string{
		"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos", "deleted_at",
	})
	accounts.AddRow(equityID, userID, "Opening", utils.AccountTypeEquity, "CNY", -1, 0, nil)
	accounts.AddRow(assetID, userID, "Cash", utils.AccountTypeAsset, "CNY", 0, 999_999_999, nil)

	transactions := sqlmock.NewRows([]string{
		"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type", "deleted_at",
	})
	transactions.AddRow(uuid.New().String(), userID, equityID, assetID, "CNY", 1, 0, utils.TransactionTypeOpeningBalance, nil)

	return accounts, transactions
}

func TestStartupReconcilePassesEmptyLedger(t *testing.T) {
	mock, _ := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	testutil.MockMeta(mock, "accounts", []string{
		"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos", "deleted_at",
	})
	testutil.MockMeta(mock, "transactions", []string{
		"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type", "deleted_at",
	})

	accounts, transactions := emptyLedgerRows()
	expectReadOnlyLedgerQueries(mock, accounts, transactions)
	mock.ExpectCommit()

	err := StartupReconcile(t.Context())
	if err != nil {
		t.Fatalf("StartupReconcile should pass on an empty ledger: %v", err)
	}
	if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
		t.Fatalf("expectations failed: %v", expectationErr)
	}
}

func TestStartupReconcileOneNanoBoundaryBlocksProductionOnly(t *testing.T) {
	for _, mode := range []struct {
		name string
		env  map[string]string
		want bool // want error returned
	}{
		{name: "production defaults to block", env: map[string]string{"GF_ENV": "production"}, want: true},
		{name: "non-production warn continues", env: map[string]string{EnvStartupReconciliation: "warn"}, want: false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			mock, _ := testutil.InitMockDB(t)
			mock.MatchExpectationsInOrder(false)
			testutil.MockDBInit(mock)
			for key, value := range mode.env {
				t.Setenv(key, value)
			}

			accounts, transactions := oneNanoLedgerRows()
			expectReadOnlyLedgerQueries(mock, accounts, transactions)
			mock.ExpectCommit()

			err := StartupReconcile(t.Context())
			if gotErr := err != nil; gotErr != mode.want {
				t.Fatalf("StartupReconcile returned %v; want error=%v", err, mode.want)
			}
			if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
				t.Fatalf("expectations failed: %v", expectationErr)
			}
		})
	}
}

func TestStartupReconcileWarnOverrideContinuesInProduction(t *testing.T) {
	mock, _ := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	t.Setenv("GF_ENV", "production")
	t.Setenv(EnvStartupReconciliation, "warn")

	accounts, transactions := oneNanoLedgerRows()
	expectReadOnlyLedgerQueries(mock, accounts, transactions)
	mock.ExpectCommit()

	if err := StartupReconcile(t.Context()); err != nil {
		t.Fatalf("warn override should continue boot in production: %v", err)
	}
	if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
		t.Fatalf("expectations failed: %v", expectationErr)
	}
}

func TestStartupReconcileOffSkipsDatabase(t *testing.T) {
	mock, _ := testutil.InitMockDB(t)
	testutil.MockDBInit(mock)
	t.Setenv("GF_ENV", "production")
	t.Setenv(EnvStartupReconciliation, "off")

	if err := StartupReconcile(t.Context()); err != nil {
		t.Fatalf("off mode should never fail boot: %v", err)
	}
	if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
		t.Fatalf("no database query may be issued in off mode: %v", expectationErr)
	}
}

func TestStartupReconcileDatabaseFailure(t *testing.T) {
	for _, mode := range []struct {
		name string
		env  map[string]string
		want bool // want error returned
	}{
		{name: "production block returns error", env: map[string]string{"GF_ENV": "production"}, want: true},
		{name: "non-production warn continues", env: nil, want: false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			mock, _ := testutil.InitMockDB(t)
			mock.MatchExpectationsInOrder(false)
			testutil.MockDBInit(mock)
			for key, value := range mode.env {
				t.Setenv(key, value)
			}

			mock.ExpectBegin()
			mock.ExpectExec(`SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(`SELECT .* FROM "?accounts"?`).WillReturnError(errors.New("simulated database outage"))
			mock.ExpectRollback()

			err := StartupReconcile(t.Context())
			if gotErr := err != nil; gotErr != mode.want {
				t.Fatalf("StartupReconcile returned %v; want error=%v", err, mode.want)
			}
			if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
				t.Fatalf("expectations failed: %v", expectationErr)
			}
		})
	}
}

func TestStartupReconcileUnknownMode(t *testing.T) {
	t.Run("production fails closed", func(t *testing.T) {
		mock, _ := testutil.InitMockDB(t)
		testutil.MockDBInit(mock)
		t.Setenv("GF_ENV", "production")
		t.Setenv(EnvStartupReconciliation, "maybe")

		if err := StartupReconcile(t.Context()); err == nil {
			t.Fatal("unknown mode in production must fail closed")
		}
		if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
			t.Fatalf("no database query may be issued before the env var is validated: %v", expectationErr)
		}
	})

	t.Run("non-production falls back to warn", func(t *testing.T) {
		mock, _ := testutil.InitMockDB(t)
		mock.MatchExpectationsInOrder(false)
		testutil.MockDBInit(mock)
		t.Setenv(EnvStartupReconciliation, "maybe")

		accounts, transactions := oneNanoLedgerRows()
		expectReadOnlyLedgerQueries(mock, accounts, transactions)
		mock.ExpectCommit()

		if err := StartupReconcile(t.Context()); err != nil {
			t.Fatalf("unknown mode in non-production should fall back to warn: %v", err)
		}
		if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
			t.Fatalf("expectations failed: %v", expectationErr)
		}
	})
}
