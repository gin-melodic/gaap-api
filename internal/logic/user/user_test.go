package user_test

import (
	"context"
	"testing"

	_ "gaap-api/internal/logic/user"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/test/gtest"
	"github.com/google/uuid"
)

func Test_User_Suite(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, "1")

		// Mock version check once for the suite
		testutil.MockDBInit(mock)

		// 1. GetUserProfile
		// Expectation for GetUserProfile
		// GoFrame fetches users metadata first
		testutil.MockMeta(mock, "users", []string{"id", "email", "nickname", "avatar", "plan", "theme_id", "main_currency", "created_at", "updated_at", "deleted_at"})

		// It selects from users table.
		rows := sqlmock.NewRows([]string{"id", "email", "nickname", "avatar", "plan", "created_at", "updated_at", "deleted_at"}).
			AddRow("1", "test@example.com", "Test User", "", "FREE", "2023-01-01", "2023-01-01", nil)

		mock.ExpectQuery("SELECT .* FROM \"?users\"?").WillReturnRows(rows)

		out, err := service.User().GetUserProfile(ctx)
		g.AssertNil(err)
		if out != nil {
			g.Assert(out.Email, "test@example.com")
			g.Assert(out.Nickname, "Test User")
		} else {
			g.Error("User profile should not be nil")
		}

		// 2. UpdateUserProfile
		// Expectation for UpdateUserProfile
		// Note: gdb has already cached users metadata from step 1, no need for MockMeta again

		// It updates users table.
		// gdb updates nickname, avatar, plan, updated_at + WHERE id = 1 (5 args)
		mock.ExpectExec("UPDATE \"?users\"? SET").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Then it calls GetUserProfile to return updated profile
		// GetUserProfile now uses WHERE id = 1 (1 arg)
		rows = sqlmock.NewRows([]string{"id", "email", "nickname", "avatar", "plan", "created_at", "updated_at", "deleted_at"}).
			AddRow("1", "test@example.com", "Test User", "", "FREE", "2023-01-01", "2023-01-01", nil)
		mock.ExpectQuery("SELECT .* FROM \"?users\"?").
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(rows)

		in := model.UserUpdateInput{
			Nickname: "Test User",
		}
		out, err = service.User().UpdateUserProfile(ctx, in)
		g.AssertNil(err)
		if out != nil {
			g.Assert(out.Nickname, "Test User")
		}
		// 3. UpdateThemePreference
		inTheme := model.Theme{
			Id:   uuid.New(),
			Name: "dark",
		}

		// 3.1 Check if theme exists
		// Themes metadata check (first time access)
		testutil.MockMeta(mock, "themes", []string{"id", "name", "is_dark", "colors", "created_at", "updated_at", "deleted_at"})

		// Verify theme query
		rows = sqlmock.NewRows([]string{"id", "name", "is_dark", "colors", "created_at", "updated_at", "deleted_at"}).
			AddRow(uuid.New(), "dark", true, "{}", "2023-01-01", "2023-01-01", nil)

		mock.ExpectQuery("SELECT .* FROM \"?themes\"?").
			WithArgs(inTheme.Id).
			WillReturnRows(rows)

		// 3.2 Update user preference
		// Users meta is already cached by Step 1 & 2, so gdb won't query it again.

		// verify users update
		mock.ExpectExec("UPDATE \"?users\"? SET").
			WithArgs(inTheme.Id, sqlmock.AnyArg(), sqlmock.AnyArg()). // theme_id, updated_at, id
			WillReturnResult(sqlmock.NewResult(1, 1))

		outTheme, err := service.User().UpdateThemePreference(ctx, inTheme)
		g.AssertNil(err)
		g.Assert(outTheme.Name, "dark")
		g.Assert(outTheme.Id, uuid.New())
	})
}
