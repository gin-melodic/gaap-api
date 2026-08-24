package balance

import (
	"errors"
	"testing"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestApplyTransactionRollsBackWhenSecondAccountUpdateFails(t *testing.T) {
	mock, _ := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	testutil.MockMeta(mock, "accounts", []string{
		"id", "user_id", "balance_units", "balance_nanos", "currency_code", "deleted_at",
	})

	fromID := uuid.New()
	toID := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .* FROM "?accounts"?.*FOR UPDATE`).
		WithArgs(fromID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "balance_units", "balance_nanos", "currency_code", "deleted_at",
		}).AddRow(fromID, uuid.New(), 100, 0, "CNY", nil))
	mock.ExpectExec(`UPDATE "?accounts"? SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM "?accounts"?.*FOR UPDATE`).
		WithArgs(toID).
		WillReturnError(errors.New("injected second account failure"))
	mock.ExpectRollback()

	err := New().ApplyTransaction(t.Context(), &model.TransactionCreateInput{
		FromAccountId: fromID,
		ToAccountId:   toID,
		BalanceUnits:  10,
		CurrencyCode:  "CNY",
		Type:          utils.TransactionTypeExpense,
	})
	if err == nil {
		t.Fatal("expected transaction to fail")
	}
	if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
		t.Fatalf("transaction did not roll back as expected: %v", expectationErr)
	}
}
