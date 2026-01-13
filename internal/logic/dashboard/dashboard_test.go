package dashboard_test

import (
	"context"
	"testing"
	"time"

	_ "gaap-api/internal/logic/dashboard"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/middleware"
	"gaap-api/internal/service"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/test/gtest"
	"github.com/google/uuid"
)

func Test_Dashboard_GetDashboardSummary(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		testutil.MockDBInit(mock)
		userId := uuid.New().String()
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, userId)

		// Expectation for Assets (SELECT * FROM accounts WHERE type=Asset)
		testutil.MockMeta(mock, "accounts", []string{"id", "balance_units", "balance_nanos", "currency_code", "type", "is_group"})

		rows := sqlmock.NewRows([]string{"id", "balance_units", "balance_nanos", "currency_code", "type", "is_group", "user_id"}).
			AddRow(uuid.New().String(), 1000, 0, "CNY", utils.AccountTypeAsset, false, userId).
			AddRow(uuid.New().String(), 500, 500_000_000, "CNY", utils.AccountTypeAsset, false, userId)

		mock.ExpectQuery("SELECT .* FROM \"?accounts\"?.*").
			WithArgs(userId, utils.AccountTypeAsset, false).
			WillReturnRows(rows)

		// Expectation for Liabilities
		lRows := sqlmock.NewRows([]string{"id", "balance_units", "balance_nanos", "currency_code", "type", "is_group", "user_id"}).
			AddRow(uuid.New().String(), 300, 0, "CNY", utils.AccountTypeLiability, false, userId)

		mock.ExpectQuery("SELECT .* FROM \"?accounts\"?.*").
			WithArgs(userId, utils.AccountTypeLiability, false).
			WillReturnRows(lRows)

		out, err := service.Dashboard().GetDashboardSummary(ctx)
		g.AssertNil(err)
		g.AssertNE(out, nil)
		// Assets: 1000 + 500.5 = 1500.5
		g.Assert(out.AssetsUnits, int64(1500))
		g.Assert(out.AssetsNanos, 500_000_000)

		// Liabilities: 300
		g.Assert(out.LiabilitiesUnits, int64(300))
		g.Assert(out.LiabilitiesNanos, 0)

		// Net Worth: 1500.5 - 300 = 1200.5
		g.Assert(out.NetWorthUnits, int64(1200))
		g.Assert(out.NetWorthNanos, 500_000_000)
	})
}

func Test_Dashboard_GetMonthlyStats(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		userId := uuid.New().String()
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, userId)

		// GoFrame executes metadata queries first for the transactions table
		testutil.MockMeta(mock, "transactions", []string{"id", "balance_units", "balance_nanos", "currency_code", "type", "date"})

		// Expectation for Income
		iRows := sqlmock.NewRows([]string{"id", "balance_units", "balance_nanos", "currency_code", "type", "user_id", "date"}).
			AddRow(uuid.New().String(), 2000, 0, "CNY", utils.TransactionTypeIncome, userId, time.Now())

		mock.ExpectQuery("SELECT .* FROM \"?transactions\"?.*").
			WithArgs(userId, utils.TransactionTypeIncome, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(iRows)

		// Expectation for Expense
		eRows := sqlmock.NewRows([]string{"id", "balance_units", "balance_nanos", "currency_code", "type", "user_id", "date"}).
			AddRow(uuid.New().String(), 800, 0, "CNY", utils.TransactionTypeExpense, userId, time.Now())

		mock.ExpectQuery("SELECT .* FROM \"?transactions\"?.*").
			WithArgs(userId, utils.TransactionTypeExpense, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(eRows)

		out, err := service.Dashboard().GetMonthlyStats(ctx)
		g.AssertNil(err)
		g.AssertNE(out, nil)
		g.Assert(out.IncomeUnits, int64(2000))
		g.Assert(out.ExpenseUnits, int64(800))
	})
}
