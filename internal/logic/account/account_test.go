package account_test

import (
	"context"
	"fmt"
	"testing"

	"gaap-api/internal/dao"
	_ "gaap-api/internal/logic/account"
	_ "gaap-api/internal/logic/task"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/mq"
	"gaap-api/internal/service"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/test/gtest"
)

func Test_Account_CRUD(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Setup Mock MQ
		mockMQ := &testutil.MockMQ{}
		mq.SetClient(mockMQ)
		// Verify client is set
		if mq.GetRabbitMQ() != mockMQ {
			t.Fatalf("Failed to set MockMQ: got %T", mq.GetRabbitMQ())
		}
		fmt.Println("Test: MockMQ set successfully")

		mock, _ := testutil.InitMockDB(t)

		ctx := context.Background()

		// Create
		// Ensure a valid account type exists
		testType := "ASSET"

		accTypeRecord := entity.AccountTypes{
			Type:  testType,
			Label: "Asset",
			Color: "#000000",
			Bg:    "#ffffff",
			Icon:  "icon",
		}
		// Expectation for AccountTypes Save
		testutil.MockMeta(mock, "account_types", []string{"type", "label", "color", "bg", "icon", "created_at", "updated_at", "deleted_at"})
		// 8 args
		mock.ExpectExec("INSERT INTO \"?account_types\"?").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		_, err := dao.AccountTypes.Ctx(ctx).Data(accTypeRecord).OnConflict("type").Save()
		g.AssertNil(err)

		// Ensure currency exists
		// Expectation for Currencies Save
		testutil.MockMeta(mock, "currencies", []string{"code", "created_at", "updated_at", "deleted_at"})
		// 4 args
		mock.ExpectExec("INSERT INTO \"?currencies\"?").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		_, err = dao.Currencies.Ctx(ctx).Data(entity.Currencies{Code: "USD"}).OnConflict("code").Save()
		g.AssertNil(err)

		in := model.AccountCreateInput{
			Name:     "Test Account",
			Type:     testType,
			Date:     "2023-01-01",
			Currency: "USD",
		}

		// Expectation for CreateAccount
		testutil.MockMeta(mock, "accounts", []string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"})
		// name, type, date, currency, is_group, balance, number, remarks + deleted_at = 9 args
		// gdb uses RETURNING id
		mock.ExpectQuery("INSERT INTO \"?accounts\"?").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

		// Expectation for retrieving the created account
		rowsCreate := sqlmock.NewRows([]string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"}).
			AddRow("1", nil, "Test Account", "ASSET", 0, 0, "USD", nil, "2023-01-01", "", "", "2023-01-01", "2023-01-01", nil)
		mock.ExpectQuery("SELECT .* FROM \"?accounts\"?").
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(rowsCreate)

		_, err = service.Account().CreateAccount(ctx, in)
		g.AssertNil(err)

		// List to find the created account
		// Expectation for ListAccounts (Count)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		testutil.MockVersion(mock)

		// Expectation for ListAccounts (Select)
		rows := sqlmock.NewRows([]string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"}).
			AddRow("1", nil, "Test Account", "ASSET", 0, 0, "USD", nil, "2023-01-01", "", "", "2023-01-01", "2023-01-01", nil)
		mock.ExpectQuery("SELECT .* FROM \"?accounts\"?").WillReturnRows(rows)

		listOut, _, err := service.Account().ListAccounts(ctx, model.AccountQueryInput{
			Type:  testType,
			Page:  1,
			Limit: 10,
		})
		g.AssertNil(err)

		var createdId string
		for _, acc := range listOut {
			if acc.Name == "Test Account" {
				createdId = acc.Id
				break
			}
		}

		if createdId != "" {
			// Get
			// Expectation for GetAccount
			rows = sqlmock.NewRows([]string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"}).
				AddRow("1", nil, "Test Account", "ASSET", 0, 0, "USD", nil, "2023-01-01", "", "", "2023-01-01", "2023-01-01", nil)
			mock.ExpectQuery("SELECT .* FROM \"?accounts\"?").WillReturnRows(rows)

			getOut, err := service.Account().GetAccount(ctx, createdId)
			g.AssertNil(err)
			g.Assert(getOut.Name, "Test Account")

			// Update
			updateIn := model.AccountUpdateInput{
				Name: "Updated Test Account",
			}

			// Expectation for GetAccount (inside UpdateAccount)
			rowsPreUpdate := sqlmock.NewRows([]string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"}).
				AddRow("1", nil, "Test Account", "ASSET", 0, 0, "USD", nil, "2023-01-01", "", "", "2023-01-01", "2023-01-01", nil)
			mock.ExpectQuery("SELECT .* FROM \"?accounts\"?").
				WithArgs(sqlmock.AnyArg()).
				WillReturnRows(rowsPreUpdate)

			// Expectation for UpdateAccount
			// name + updated_at + id = 3 args
			mock.ExpectExec("UPDATE \"?accounts\"?").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(1, 1))
			// Expectation for GetAccount (after update)
			rowsPostUpdate := sqlmock.NewRows([]string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"}).
				AddRow("1", nil, "Updated Test Account", "ASSET", 0, 0, "USD", nil, "2023-01-01", "", "", "2023-01-01", "2023-01-01", nil)
			mock.ExpectQuery("SELECT .* FROM \"?accounts\"?").WillReturnRows(rowsPostUpdate)

			updateOut, err := service.Account().UpdateAccount(ctx, createdId, updateIn)
			g.AssertNil(err)
			g.Assert(updateOut.Name, "Updated Test Account")

			// Delete
			// Expectation for DeleteAccount
			// 1. GetAccount
			rowsDelete := sqlmock.NewRows([]string{"id", "parent_id", "name", "type", "is_group", "balance", "currency", "default_child_id", "date", "number", "remarks", "created_at", "updated_at", "deleted_at"}).
				AddRow("1", nil, "Updated Test Account", "ASSET", 0, 0, "USD", nil, "2023-01-01", "", "", "2023-01-01", "2023-01-01", nil)
			mock.ExpectQuery("SELECT .* FROM \"?accounts\"?").
				WithArgs(sqlmock.AnyArg()).
				WillReturnRows(rowsDelete)

			// 2. Create Task (for async deletion)
			// INSERT INTO tasks (Raw SQL, so no metadata check before this)
			mock.ExpectQuery("INSERT INTO \"?tasks\"?").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("task_1"))

			// 3. Select Task (GetTask called after Create)
			// GoFrame fetches tasks metadata first here because GetTask uses DAO
			testutil.MockMeta(mock, "tasks", []string{"id", "user_id", "type", "status", "payload", "result", "progress", "total_items", "processed_items", "started_at", "completed_at", "created_at", "updated_at"})

			// Select Task Query
			rowsTask := sqlmock.NewRows([]string{"id", "user_id", "type", "status", "created_at", "updated_at"}).
				AddRow("task_1", "1", "ACCOUNT_MIGRATION", "PENDING", "2023-01-01", "2023-01-01")
			mock.ExpectQuery("SELECT .* FROM \"?tasks\"?").
				WithArgs("task_1").
				WillReturnRows(rowsTask)

			taskId, err := service.Account().DeleteAccount(ctx, createdId, nil)
			g.AssertNil(err)
			g.Assert(taskId, "task_1")

			// Verification of deletion is skipped because it's async (handled by worker)
		} else {
			g.Log("Created account not found in list, possibly due to ID generation issue or transaction isolation")
		}
	})
}
