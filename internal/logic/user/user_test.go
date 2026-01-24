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

// Test_User_Suite tests user service methods.
// Note: Cache layer automatically falls back to DB when Redis is unavailable,
// so these tests work without mocking Redis.
func Test_User_Suite(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		userId := uuid.New().String()
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, userId)

		// Mock version check once for the suite
		testutil.MockDBInit(mock)

		// 1. GetUserProfile
		// Note: Cache fallback to DB - expects DB query since Redis unavailable in test
		// Expectation for GetUserProfile
		// GoFrame fetches users metadata first
		testutil.MockMeta(mock, "users", []string{"id", "email", "nickname", "avatar", "plan", "theme_id", "main_currency", "created_at", "updated_at", "deleted_at"})

		// It selects from users table.
		rows := sqlmock.NewRows([]string{"id", "email", "nickname", "avatar", "plan", "created_at", "updated_at", "deleted_at"}).
			AddRow(userId, "test@example.com", "Test User", "", 0, "2023-01-01", "2023-01-01", nil)

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
		// Note: Cache invalidation is async and non-blocking, so it doesn't affect test expectations
		// Expectation for UpdateUserProfile
		// Note: gdb has already cached users metadata from step 1, no need for MockMeta again

		// It updates users table.
		// gdb updates nickname, avatar, plan, main_currency, updated_at + WHERE id = userId (6 args)
		mock.ExpectExec("UPDATE \"?users\"? SET").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Then it calls GetUserProfile to return updated profile
		// Cache miss due to invalidation, will query DB again
		// GetUserProfile now uses WHERE id = userId (1 arg)
		rows = sqlmock.NewRows([]string{"id", "email", "nickname", "avatar", "plan", "created_at", "updated_at", "deleted_at"}).
			AddRow(userId, "test@example.com", "Test User", "", 0, "2023-01-01", "2023-01-01", nil)
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
			AddRow(inTheme.Id.String(), "dark", true, "{}", "2023-01-01", "2023-01-01", nil)

		mock.ExpectQuery("SELECT .* FROM \"?themes\"?").
			WithArgs(inTheme.Id).
			WillReturnRows(rows)

		// 3.2 Update user preference
		// Users meta is already cached by Step 1 & 2, so gdb won't query it again.
		// Note: Cache invalidation is async and non-blocking

		// verify users update
		mock.ExpectExec("UPDATE \"?users\"? SET").
			WithArgs(inTheme.Id, sqlmock.AnyArg(), userId). // theme_id, updated_at, id
			WillReturnResult(sqlmock.NewResult(1, 1))

		outTheme, err := service.User().UpdateThemePreference(ctx, inTheme)
		g.AssertNil(err)
		g.Assert(outTheme.Name, "dark")
		g.Assert(outTheme.Id, inTheme.Id)
	})
}
