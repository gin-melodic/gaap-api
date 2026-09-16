package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/guid"
)

// newOpsTestServer boots an ephemeral server with the given middleware chain in
// front of a handler that sets UserIdKey like AuthMiddleware does.
func newOpsTestServer(t *testing.T, chain ...func(r *ghttp.Request)) *ghttp.Server {
	t.Helper()
	s := g.Server(guid.S())
	s.SetAddr("127.0.0.1:0")
	s.SetDumpRouterMap(false)

	group := s.Group("/")
	// Set the session identity first, like AuthMiddleware does.
	group.Middleware(func(r *ghttp.Request) {
		r.SetCtx(context.WithValue(r.Context(), UserIdKey, "user-1"))
		r.Middleware.Next()
	})
	for _, mw := range chain {
		group.Middleware(mw)
	}
	group.GET("/ops-test", func(r *ghttp.Request) {
		r.Response.WriteJson(g.Map{"ok": true})
	})

	s.SetDumpRouterMap(false)
	if err := s.Start(); err != nil {
		t.Fatalf("server start failed: %v", err)
	}
	t.Cleanup(func() { s.Shutdown() })
	return s
}

func getOps(t *testing.T, s *ghttp.Server, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/ops-test", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestResolveOpsPathCode(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv(EnvOpsPathCode, "")
		code, err := ResolveOpsPathCode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != DefaultOpsPathCode {
			t.Fatalf("code = %q, want %q", code, DefaultOpsPathCode)
		}
	})
	t.Run("valid override", func(t *testing.T) {
		t.Setenv(EnvOpsPathCode, "  Xy9Zw2 ")
		code, err := ResolveOpsPathCode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code != "xy9zw2" {
			t.Fatalf("code = %q, want xy9zw2", code)
		}
	})
	t.Run("invalid override fails", func(t *testing.T) {
		for _, bad := range []string{"UP", "a", "has-dash", "has space", "12345678901234567"} {
			t.Setenv(EnvOpsPathCode, bad)
			if _, err := ResolveOpsPathCode(); err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		}
	})
}

func initOpsAdminTest(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	testutil.ResetSchemaMocks()
	mock, _ := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	testutil.MockMeta(mock, "users", []string{
		"id", "password", "email", "nickname", "avatar", "plan", "theme_id", "main_currency",
		"two_factor_secret", "two_factor_enabled", "created_at", "updated_at", "deleted_at",
	})
	return mock
}

func TestOpsAdminUnsetAllowlistDeniesEveryone(t *testing.T) {
	mock := initOpsAdminTest(t)
	t.Setenv(EnvOpsAdminEmails, "")
	t.Setenv(EnvOpsAPIToken, "")
	mock.ExpectQuery(`SELECT .* FROM .users. WHERE .*id.*`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("admin@gaap.local"))

	s := newOpsTestServer(t, OpsAdmin)
	if rec := getOps(t, s, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestOpsAdminListedEmailIsAllowed(t *testing.T) {
	mock := initOpsAdminTest(t)
	t.Setenv(EnvOpsAdminEmails, "other@example.com, Admin@Gaap.local ")
	t.Setenv(EnvOpsAPIToken, "")
	mock.ExpectQuery(`SELECT .* FROM .users. WHERE .*id.*`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("ADMIN@GAAP.LOCAL"))

	s := newOpsTestServer(t, OpsAdmin)
	if rec := getOps(t, s, nil); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestOpsAdminUnlistedEmailIsDenied(t *testing.T) {
	mock := initOpsAdminTest(t)
	t.Setenv(EnvOpsAdminEmails, "someone-else@example.com")
	t.Setenv(EnvOpsAPIToken, "")
	mock.ExpectQuery(`SELECT .* FROM .users. WHERE .*id.*`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("admin@gaap.local"))

	s := newOpsTestServer(t, OpsAdmin)
	if rec := getOps(t, s, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestOpsAdminLookupFailureIs500(t *testing.T) {
	mock := initOpsAdminTest(t)
	t.Setenv(EnvOpsAdminEmails, "admin@gaap.local")
	t.Setenv(EnvOpsAPIToken, "")
	mock.ExpectQuery(`SELECT .* FROM .users. WHERE .*id.*`).WillReturnError(fmt.Errorf("db down"))

	s := newOpsTestServer(t, OpsAdmin)
	if rec := getOps(t, s, nil); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestOpsAdminSharedSecret(t *testing.T) {
	mock := initOpsAdminTest(t)
	t.Setenv(EnvOpsAdminEmails, "admin@gaap.local")
	t.Setenv(EnvOpsAPIToken, "s3cret-token")

	// Only the final, correctly authenticated request reaches the DB.
	mock.ExpectQuery(`SELECT .* FROM .users. WHERE .*id.*`).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("admin@gaap.local"))

	s := newOpsTestServer(t, OpsAdmin)

	if rec := getOps(t, s, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("missing header: status = %d, want 403", rec.Code)
	}
	if rec := getOps(t, s, map[string]string{HeaderOpsToken: "wrong"}); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong header: status = %d, want 403", rec.Code)
	}
	if rec := getOps(t, s, map[string]string{HeaderOpsToken: "s3cret-token"}); rec.Code != http.StatusOK {
		t.Fatalf("correct header: status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestOpsRateLimit(t *testing.T) {
	opsRateMu.Lock()
	opsRateBuckets = map[string]*opsRateBucket{}
	opsRateMu.Unlock()

	s := newOpsTestServer(t, OpsRateLimit)
	headers := map[string]string{"X-Forwarded-For": "203.0.113.7"}

	for i := 0; i < opsRateLimitPerMinute; i++ {
		rec := getOps(t, s, headers)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d/%d: status = %d, want 200", i+1, opsRateLimitPerMinute, rec.Code)
		}
	}
	rec := getOps(t, s, headers)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request %d: status = %d, want 429", opsRateLimitPerMinute+1, rec.Code)
	}
	if rec.Body.String() != `{"message":"too many requests"}` {
		t.Fatalf("body = %s, want generic 429 JSON", rec.Body.String())
	}
}
