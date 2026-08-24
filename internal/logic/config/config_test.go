package config_test

import (
	"context"
	"testing"

	_ "gaap-api/internal/logic/config"
	"gaap-api/internal/service"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/test/gtest"
)

func Test_Config_Currencies(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		ctx := context.Background()

		// Add Currency
		// Expectation for AddCurrency (Insert)
		testutil.MockMeta(mock, "currencies", []string{"code", "created_at", "updated_at", "deleted_at"})
		// code + deleted_at = 2 args
		// gdb uses RETURNING code because it's PK
		mock.ExpectQuery("INSERT INTO \"?currencies\"?").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("CNY"))

		// Expectation for AddCurrency (List after insert)
		// ListCurrencies is called inside AddCurrency
		// It selects code from currencies

		testutil.MockDBInit(mock)

		rows := sqlmock.NewRows([]string{"code"}).AddRow("CNY")
		mock.ExpectQuery("SELECT \"?code\"? FROM \"?currencies\"?").WillReturnRows(rows)

		out, err := service.Config().AddCurrency(ctx, "CNY")
		g.AssertNil(err)
		g.Assert(len(out), 1)
		g.Assert(out[0], "CNY")

		// List Currencies
		// Expectation for ListCurrencies
		rows = sqlmock.NewRows([]string{"code"}).AddRow("CNY").AddRow("USD")
		mock.ExpectQuery("SELECT \"?code\"? FROM \"?currencies\"?").WillReturnRows(rows)

		listOut, err := service.Config().ListCurrencies(ctx)
		g.AssertNil(err)
		g.Assert(len(listOut), 2)
		g.Assert(listOut[0], "CNY")
		g.Assert(listOut[1], "USD")

		// Delete Currency
		// Expectation for DeleteCurrency
		// Schema cached
		// Unscoped delete -> DELETE FROM
		// code = 1 arg
		mock.ExpectExec("DELETE FROM \"?currencies\"?").
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = service.Config().DeleteCurrency(ctx, "CNY")
		g.AssertNil(err)
	})
}
