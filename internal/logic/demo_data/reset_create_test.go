package demo_data

import (
	"context"
	"os"
	"testing"

	"gaap-api/internal/dao"
	"gaap-api/internal/model/entity"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

func TestDemoEnsureBaselineCreatesMissingUser(t *testing.T) {
	link := os.Getenv("DEMO_RESET_INTEGRATION_DATABASE_LINK")
	if link == "" {
		t.Skip("set DEMO_RESET_INTEGRATION_DATABASE_LINK to run PostgreSQL reset integration test")
	}
	// Type is set explicitly so the link may be given with or without a driver prefix.
	if err := gdb.SetConfigGroup("default", gdb.ConfigGroup{gdb.ConfigNode{Type: "pgsql", Link: link}}); err != nil {
		t.Fatalf("set database config group: %v", err)
	}
	ctx := t.Context()

	password := "auto-created-demo-password"
	email := "auto-created-" + uuid.New().String() + "@example.com"

	userColumns := dao.Users.Columns()
	accountColumns := dao.Accounts.Columns()
	currencyColumns := dao.Currencies.Columns()
	baselineColumns := dao.DemoUserBaselines.Columns()

	if _, err := dao.Currencies.Ctx(ctx).Data(g.Map{currencyColumns.Code: defaultDemoBaseCurrency}).InsertIgnore(); err != nil {
		t.Fatalf("insert currency: %v", err)
	}
	t.Setenv("ONLINE_DEMO_USER_EMAIL", email)
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", password)
	config, err := LoadConfig(ctx)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	service := New()
	if err := service.ensureBaseline(ctx, config); err != nil {
		t.Fatalf("ensure baseline with missing user: %v", err)
	}
	var createdID uuid.UUID
	defer func() {
		if createdID == uuid.Nil {
			return
		}
		cleanups := []struct {
			name string
			del  func() error
		}{
			{"demo_user_baselines", func() error { _, err := dao.DemoUserBaselines.Ctx(ctx).Unscoped().Where(baselineColumns.UserId, createdID).Delete(); return err }},
			{"users", func() error { _, err := dao.Users.Ctx(ctx).Unscoped().Where(userColumns.Id, createdID).Delete(); return err }},
		}
		for _, c := range cleanups {
			if err := c.del(); err != nil {
				t.Errorf("cleanup %s: %v", c.name, err)
			}
		}
	}()

	var created entity.Users
	if err := dao.Users.Ctx(ctx).Where(userColumns.Email, email).Scan(&created); err != nil {
		t.Fatalf("load auto-created demo user: %v", err)
	}
	if created.Id == uuid.Nil {
		t.Fatal("demo user was not created")
	}
	createdID = created.Id

	accountCount, err := dao.Accounts.Ctx(ctx).Where(accountColumns.UserId, created.Id).Count()
	if err != nil || accountCount != len(demoAccountSeeds) {
		t.Fatalf("auto-created demo accounts = %d, want %d: err=%v", accountCount, len(demoAccountSeeds), err)
	}
	for _, seed := range demoAccountSeeds {
		var account entity.Accounts
		if err := dao.Accounts.Ctx(ctx).Where(accountColumns.UserId, created.Id).
			Where(accountColumns.Name, seed.name).Scan(&account); err != nil || account.Id == uuid.Nil {
			t.Fatalf("seeded demo account %q missing: err=%v", seed.name, err)
		}
	}
	var checking entity.Accounts
	err = dao.Accounts.Ctx(ctx).Where(accountColumns.UserId, created.Id).
		Where(accountColumns.Name, "Checking").Scan(&checking)
	if err != nil {
		t.Fatalf("load asset account: %v", err)
	}
	if checking.ParentId == uuid.Nil || checking.EquityAccountId == uuid.Nil {
		t.Fatalf("asset account relationships were not wired: %+v", checking)
	}
	var baseline entity.DemoUserBaselines
	err = dao.DemoUserBaselines.Ctx(ctx).Where(baselineColumns.UserId, created.Id).Scan(&baseline)
	if err != nil || baseline.UserId == uuid.Nil {
		t.Fatalf("demo baseline was not captured for the new user: %+v err=%v", baseline, err)
	}

	// A second run must keep the existing row (no duplicate creation, no error).
	if err := service.ensureBaseline(ctx, config); err != nil {
		t.Fatalf("second ensure baseline should be a no-op: %v", err)
	}
	againCount, _ := dao.Users.Ctx(ctx).Where(userColumns.Email, email).Unscoped().Count()
	if againCount != 1 {
		t.Fatalf("duplicate user rows = %d, want 1", againCount)
	}
}

var _ context.Context
