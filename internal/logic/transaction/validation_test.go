package transaction

import (
	"testing"

	"gaap-api/internal/logic/utils"
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
