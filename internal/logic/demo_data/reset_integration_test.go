package demo_data

import (
	"os"
	"testing"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/model/entity"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestDemoResetRestoresBaselineAndLeavesOtherUsersUntouched(t *testing.T) {
	link := os.Getenv("DEMO_RESET_INTEGRATION_DATABASE_LINK")
	if link == "" {
		t.Skip("set DEMO_RESET_INTEGRATION_DATABASE_LINK to run PostgreSQL reset integration test")
	}
	gdb.SetConfigGroup("default", gdb.ConfigGroup{gdb.ConfigNode{Link: link}})
	ctx := t.Context()

	password := "integration-demo-password"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	demoUserID := uuid.New()
	otherUserID := uuid.New()
	parentID := uuid.New()
	assetID := uuid.New()
	equityID := uuid.New()
	transactionID := uuid.New()
	otherAccountID := uuid.New()
	visitorAccountID := uuid.New()

	userColumns := dao.Users.Columns()
	accountColumns := dao.Accounts.Columns()
	transactionColumns := dao.Transactions.Columns()
	runColumns := dao.DemoDataGenerationRuns.Columns()
	baselineColumns := dao.DemoUserBaselines.Columns()
	currencyColumns := dao.Currencies.Columns()
	accountTypeColumns := dao.AccountTypes.Columns()

	_, _ = dao.Currencies.Ctx(ctx).Data(g.Map{currencyColumns.Code: "USD"}).InsertIgnore()
	for accountType, label := range map[int]string{1: "Asset", 5: "Equity"} {
		_, _ = dao.AccountTypes.Ctx(ctx).Data(g.Map{accountTypeColumns.Type: accountType, accountTypeColumns.Label: label}).InsertIgnore()
	}
	for _, user := range []g.Map{
		{userColumns.Id: demoUserID, userColumns.Password: string(passwordHash), userColumns.Email: "demo-reset-" + demoUserID.String() + "@example.com", userColumns.Nickname: "Baseline Demo", userColumns.Plan: 1, userColumns.MainCurrency: "USD", userColumns.TwoFactorEnabled: false},
		{userColumns.Id: otherUserID, userColumns.Password: string(passwordHash), userColumns.Email: "other-reset-" + otherUserID.String() + "@example.com", userColumns.Nickname: "Other User", userColumns.Plan: 1, userColumns.MainCurrency: "USD", userColumns.TwoFactorEnabled: false},
	} {
		if _, err := dao.Users.Ctx(ctx).Data(user).Insert(); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = dao.DemoUserBaselines.Ctx(ctx).Unscoped().Where(baselineColumns.UserId, demoUserID).Delete()
		_, _ = dao.Transactions.Ctx(ctx).Unscoped().Where(transactionColumns.UserId, demoUserID).Delete()
		_, _ = dao.Accounts.Ctx(ctx).Unscoped().Where(accountColumns.UserId, demoUserID).Delete()
		_, _ = dao.Accounts.Ctx(ctx).Unscoped().Where(accountColumns.UserId, otherUserID).Delete()
		_, _ = dao.DemoDataGenerationRuns.Ctx(ctx).Unscoped().Where(runColumns.UserId, demoUserID).Delete()
		_, _ = dao.Users.Ctx(ctx).Unscoped().Where(userColumns.Id, []uuid.UUID{demoUserID, otherUserID}).Delete()
	})

	accountRows := []g.Map{
		{accountColumns.Id: parentID, accountColumns.UserId: demoUserID, accountColumns.Name: "Assets", accountColumns.Type: 1, accountColumns.IsGroup: true, accountColumns.CurrencyCode: "USD", accountColumns.BalanceUnits: int64(0), accountColumns.BalanceNanos: 0},
		{accountColumns.Id: assetID, accountColumns.UserId: demoUserID, accountColumns.Name: "Checking", accountColumns.Type: 1, accountColumns.IsGroup: false, accountColumns.CurrencyCode: "USD", accountColumns.BalanceUnits: int64(9223372036854770000), accountColumns.BalanceNanos: 999999999},
		{accountColumns.Id: equityID, accountColumns.UserId: demoUserID, accountColumns.Name: "Opening Equity", accountColumns.Type: 5, accountColumns.IsGroup: false, accountColumns.CurrencyCode: "USD", accountColumns.BalanceUnits: int64(0), accountColumns.BalanceNanos: 0},
		{accountColumns.Id: otherAccountID, accountColumns.UserId: otherUserID, accountColumns.Name: "Other Account", accountColumns.Type: 1, accountColumns.IsGroup: false, accountColumns.CurrencyCode: "USD", accountColumns.BalanceUnits: int64(77), accountColumns.BalanceNanos: 250000000},
	}
	for _, account := range accountRows {
		if _, err := dao.Accounts.Ctx(ctx).Data(account).Insert(); err != nil {
			t.Fatalf("insert account: %v", err)
		}
	}
	if _, err := dao.Accounts.Ctx(ctx).Where(accountColumns.Id, assetID).Data(g.Map{accountColumns.ParentId: parentID, accountColumns.EquityAccountId: equityID}).Update(); err != nil {
		t.Fatalf("link asset account: %v", err)
	}
	if _, err := dao.Accounts.Ctx(ctx).Where(accountColumns.Id, parentID).Data(g.Map{accountColumns.DefaultChildId: assetID}).Update(); err != nil {
		t.Fatalf("link parent account: %v", err)
	}
	if _, err := dao.Transactions.Ctx(ctx).Data(g.Map{
		transactionColumns.Id: transactionID, transactionColumns.UserId: demoUserID,
		transactionColumns.Date: gtime.Now(), transactionColumns.FromAccountId: equityID,
		transactionColumns.ToAccountId: assetID, transactionColumns.CurrencyCode: "USD",
		transactionColumns.BalanceUnits: int64(12), transactionColumns.BalanceNanos: 345000000,
		transactionColumns.Note: "Baseline transaction", transactionColumns.Type: 1,
	}).Insert(); err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	if _, err := dao.DemoDataGenerationRuns.Ctx(ctx).Data(g.Map{
		runColumns.UserId: demoUserID, runColumns.BusinessDate: "2026-08-24", runColumns.GeneratedCount: 1,
	}).Insert(); err != nil {
		t.Fatalf("insert generation run: %v", err)
	}

	t.Setenv("ONLINE_DEMO_USER_EMAIL", "demo-reset-"+demoUserID.String()+"@example.com")
	t.Setenv("ONLINE_DEMO_USER_PASSWORD", password)
	config, err := LoadConfig(ctx)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	service := New()
	if err := service.ensureBaseline(ctx, config); err != nil {
		t.Fatalf("capture baseline: %v", err)
	}

	if _, err := dao.Users.Ctx(ctx).Where(userColumns.Id, demoUserID).Data(g.Map{userColumns.Nickname: "Visitor Edit"}).Update(); err != nil {
		t.Fatalf("mutate user: %v", err)
	}
	if _, err := dao.Accounts.Ctx(ctx).Where(accountColumns.Id, assetID).Data(g.Map{accountColumns.Name: "Visitor Account", accountColumns.BalanceUnits: int64(1), accountColumns.BalanceNanos: 1, accountColumns.DeletedAt: gtime.Now()}).Update(); err != nil {
		t.Fatalf("mutate account: %v", err)
	}
	if _, err := dao.Accounts.Ctx(ctx).Data(g.Map{
		accountColumns.Id: visitorAccountID, accountColumns.UserId: demoUserID,
		accountColumns.Name: "Visitor-created account", accountColumns.Type: 1,
		accountColumns.IsGroup: false, accountColumns.CurrencyCode: "USD",
		accountColumns.BalanceUnits: int64(42), accountColumns.BalanceNanos: 500000000,
	}).Insert(); err != nil {
		t.Fatalf("create visitor account: %v", err)
	}
	if _, err := dao.Transactions.Ctx(ctx).Where(transactionColumns.Id, transactionID).Delete(); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}

	resetDate := time.Date(2026, time.August, 26, 0, 0, 0, 0, config.Location)
	performed, err := service.resetForDate(ctx, config, resetDate)
	if err != nil || !performed {
		t.Fatalf("reset baseline: performed=%t err=%v", performed, err)
	}
	var restored entity.Accounts
	if err := dao.Accounts.Ctx(ctx).Where(accountColumns.Id, assetID).Scan(&restored); err != nil {
		t.Fatalf("load restored account: %v", err)
	}
	if restored.Name != "Checking" || restored.BalanceUnits != 9223372036854770000 || restored.BalanceNanos != 999999999 || restored.DeletedAt != nil {
		t.Fatalf("account was not restored exactly: %+v", restored)
	}
	if restored.ParentId != parentID || restored.EquityAccountId != equityID {
		t.Fatalf("account relationships were not restored: %+v", restored)
	}
	visitorCount, err := dao.Accounts.Ctx(ctx).Where(accountColumns.Id, visitorAccountID).Count()
	if err != nil || visitorCount != 0 {
		t.Fatalf("visitor-created account survived reset: count=%d err=%v", visitorCount, err)
	}
	transactionCount, err := dao.Transactions.Ctx(ctx).Where(transactionColumns.Id, transactionID).Count()
	if err != nil || transactionCount != 1 {
		t.Fatalf("transaction count after reset = %d, err=%v", transactionCount, err)
	}
	var other entity.Accounts
	if err := dao.Accounts.Ctx(ctx).Where(accountColumns.Id, otherAccountID).Scan(&other); err != nil || other.BalanceUnits != 77 || other.BalanceNanos != 250000000 {
		t.Fatalf("other user account changed: %+v err=%v", other, err)
	}
	performed, err = service.resetForDate(ctx, config, resetDate)
	if err != nil || performed {
		t.Fatalf("same-day reset should be skipped: performed=%t err=%v", performed, err)
	}
}
