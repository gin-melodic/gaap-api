package transaction

import (
	"context"
	"testing"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/google/uuid"
)

func TestValidateMoneyBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		units       int64
		nanos       int
		allowSigned bool
		wantError   bool
	}{
		{name: "smallest positive", nanos: 1},
		{name: "largest canonical nanos", nanos: 999_999_999},
		{name: "large exact amount", units: 9_000_000_000_000_000_000},
		{name: "zero", wantError: true},
		{name: "negative business transaction", units: -1, wantError: true},
		{name: "negative opening balance", units: -1, allowSigned: true},
		{name: "negative fractional opening balance", nanos: -1, allowSigned: true},
		{name: "nanos overflow", nanos: 1_000_000_000, wantError: true},
		{name: "nanos underflow", nanos: -1_000_000_000, allowSigned: true, wantError: true},
		{name: "mixed positive units negative nanos", units: 1, nanos: -1, wantError: true},
		{name: "mixed negative units positive nanos", units: -1, nanos: 1, allowSigned: true, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMoney(test.units, test.nanos, test.allowSigned)
			if (err != nil) != test.wantError {
				t.Fatalf("validateMoney(%d, %d, %t) error = %v, wantError %t", test.units, test.nanos, test.allowSigned, err, test.wantError)
			}
		})
	}
}

func TestLockOwnedAccountsUsesDeterministicRowLocks(t *testing.T) {
	mock, db := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	testutil.MockMeta(mock, "accounts", []string{
		"id", "user_id", "currency_code", "is_group", "type", "deleted_at",
	})

	userID := uuid.MustParse("10000000-0000-0000-0000-000000000000")
	lowID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	highID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .* FROM "?accounts"?.*ORDER BY "?id"? ASC.*FOR UPDATE`).
		WithArgs(lowID.String(), highID.String(), userID.String()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "currency_code", "is_group", "type", "deleted_at",
		}).
			AddRow(lowID, userID, "CNY", false, utils.AccountTypeAsset, nil).
			AddRow(highID, userID, "CNY", false, utils.AccountTypeAsset, nil))
	mock.ExpectCommit()

	err := db.Transaction(context.Background(), func(ctx context.Context, tx gdb.TX) error {
		accounts, lockErr := lockOwnedAccounts(ctx, tx, userID.String(), []uuid.UUID{highID, lowID})
		if lockErr != nil {
			return lockErr
		}
		if len(accounts) != 2 || accounts[0].Id != lowID || accounts[1].Id != highID {
			t.Fatalf("accounts were not returned in deterministic order: %+v", accounts)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("lockOwnedAccounts failed: %v", err)
	}
	if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
		t.Fatalf("expected ordered SELECT FOR UPDATE: %v", expectationErr)
	}
}

func TestValidateTransactionAccountTypes(t *testing.T) {
	tests := []struct {
		name            string
		transactionType int
		fromType        int
		toType          int
		wantError       bool
	}{
		{name: "asset to expense", transactionType: utils.TransactionTypeExpense, fromType: utils.AccountTypeAsset, toType: utils.AccountTypeExpense},
		{name: "liability to expense", transactionType: utils.TransactionTypeExpense, fromType: utils.AccountTypeLiability, toType: utils.AccountTypeExpense},
		{name: "income to asset", transactionType: utils.TransactionTypeIncome, fromType: utils.AccountTypeIncome, toType: utils.AccountTypeAsset},
		{name: "asset transfer", transactionType: utils.TransactionTypeTransfer, fromType: utils.AccountTypeAsset, toType: utils.AccountTypeLiability},
		{name: "expense mislabeled transfer", transactionType: utils.TransactionTypeTransfer, fromType: utils.AccountTypeAsset, toType: utils.AccountTypeExpense, wantError: true},
		{name: "income directly to expense", transactionType: utils.TransactionTypeIncome, fromType: utils.AccountTypeIncome, toType: utils.AccountTypeExpense, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTransactionAccountTypes(test.transactionType, test.fromType, test.toType)
			if (err != nil) != test.wantError {
				t.Fatalf("validateTransactionAccountTypes() error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}

func TestResolveTransactionSortRejectsInjection(t *testing.T) {
	column, ascending, err := resolveTransactionSort("amount", "asc")
	if err != nil || column == "" || !ascending {
		t.Fatalf("valid sort rejected: column=%q ascending=%t err=%v", column, ascending, err)
	}

	invalid := []struct {
		field string
		order string
	}{
		{field: "date desc; drop table transactions;--", order: "asc"},
		{field: "date", order: "asc nulls first;--"},
		{field: "balance_decimal", order: "desc"},
	}
	for _, test := range invalid {
		if _, _, err := resolveTransactionSort(test.field, test.order); err == nil {
			t.Fatalf("unsafe sort accepted: field=%q order=%q", test.field, test.order)
		}
	}
}
