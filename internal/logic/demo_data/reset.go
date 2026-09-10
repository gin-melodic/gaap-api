package demo_data

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type demoUserSnapshot struct {
	Password         string    `json:"password"`
	Nickname         string    `json:"nickname"`
	Avatar           string    `json:"avatar"`
	Plan             int       `json:"plan"`
	ThemeID          uuid.UUID `json:"themeId"`
	MainCurrency     string    `json:"mainCurrency"`
	TwoFactorSecret  string    `json:"twoFactorSecret"`
	TwoFactorEnabled bool      `json:"twoFactorEnabled"`
}

type demoAccountSnapshot struct {
	ID              uuid.UUID   `json:"id"`
	ParentID        uuid.UUID   `json:"parentId"`
	Name            string      `json:"name"`
	Type            int         `json:"type"`
	IsGroup         bool        `json:"isGroup"`
	CurrencyCode    string      `json:"currencyCode"`
	BalanceUnits    int64       `json:"balanceUnits"`
	BalanceNanos    int         `json:"balanceNanos"`
	DefaultChildID  uuid.UUID   `json:"defaultChildId"`
	EquityAccountID uuid.UUID   `json:"equityAccountId"`
	Date            *gtime.Time `json:"date"`
	Number          string      `json:"number"`
	Remarks         string      `json:"remarks"`
	CreatedAt       *gtime.Time `json:"createdAt"`
	UpdatedAt       *gtime.Time `json:"updatedAt"`
	DeletedAt       *gtime.Time `json:"deletedAt"`
}

type demoTransactionSnapshot struct {
	ID            uuid.UUID   `json:"id"`
	Date          *gtime.Time `json:"date"`
	FromAccountID uuid.UUID   `json:"fromAccountId"`
	ToAccountID   uuid.UUID   `json:"toAccountId"`
	CurrencyCode  string      `json:"currencyCode"`
	BalanceUnits  int64       `json:"balanceUnits"`
	BalanceNanos  int         `json:"balanceNanos"`
	Note          string      `json:"note"`
	Type          int         `json:"type"`
	CreatedAt     *gtime.Time `json:"createdAt"`
	UpdatedAt     *gtime.Time `json:"updatedAt"`
	DeletedAt     *gtime.Time `json:"deletedAt"`
}

func (s *sDemoData) ensureBaseline(ctx context.Context, config Config) error {
	userColumns := dao.Users.Columns()
	var user entity.Users
	err := dao.Users.Ctx(ctx).
		Where(userColumns.Email, config.UserEmail).
		WhereNull(userColumns.DeletedAt).
		Scan(&user)
	if err != nil && err != sql.ErrNoRows {
		return gerror.Wrap(err, "failed to load configured demo user")
	}
	if user.Id == uuid.Nil {
		created, createErr := s.createDemoUser(ctx, config)
		if createErr != nil {
			return createErr
		}
		user = *created
		g.Log().Infof(ctx, "Online demo user %s was not found; created it automatically", config.UserEmail)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(config.UserPassword)); err != nil {
		return gerror.New("configured demo user password does not match ONLINE_DEMO_USER_PASSWORD")
	}
	if user.TwoFactorEnabled {
		return gerror.New("configured demo user must have two-factor authentication disabled")
	}

	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		baselineColumns := dao.DemoUserBaselines.Columns()
		count, countErr := tx.Model(dao.DemoUserBaselines.Table()).
			Where(baselineColumns.UserId, user.Id).
			Count()
		if countErr != nil {
			return gerror.Wrap(countErr, "failed to inspect demo baseline")
		}
		if count > 0 {
			return nil
		}

		// Provision any missing seeded accounts before capturing the immutable
		// baseline so an existing demo user is not frozen on a partial chart of
		// accounts that the planner cannot generate from.
		if seedErr := seedDemoAccounts(ctx, tx, user); seedErr != nil {
			return seedErr
		}

		accounts, transactions, runs, loadErr := loadBaselineRows(ctx, tx, user.Id)
		if loadErr != nil {
			return loadErr
		}
		userJSON, marshalErr := json.Marshal(demoUserSnapshot{
			Password:         user.Password,
			Nickname:         user.Nickname,
			Avatar:           user.Avatar,
			Plan:             user.Plan,
			ThemeID:          user.ThemeId,
			MainCurrency:     user.MainCurrency,
			TwoFactorSecret:  user.TwoFactorSecret,
			TwoFactorEnabled: user.TwoFactorEnabled,
		})
		if marshalErr != nil {
			return gerror.Wrap(marshalErr, "failed to encode demo user baseline")
		}
		accountsJSON, marshalErr := json.Marshal(accounts)
		if marshalErr != nil {
			return gerror.Wrap(marshalErr, "failed to encode demo account baseline")
		}
		transactionsJSON, marshalErr := json.Marshal(transactions)
		if marshalErr != nil {
			return gerror.Wrap(marshalErr, "failed to encode demo transaction baseline")
		}
		runsJSON, marshalErr := json.Marshal(runs)
		if marshalErr != nil {
			return gerror.Wrap(marshalErr, "failed to encode demo generation baseline")
		}

		_, insertErr := tx.Model(dao.DemoUserBaselines.Table()).Data(g.Map{
			baselineColumns.UserId:                 user.Id,
			baselineColumns.UserSnapshot:           string(userJSON),
			baselineColumns.AccountsSnapshot:       string(accountsJSON),
			baselineColumns.TransactionsSnapshot:   string(transactionsJSON),
			baselineColumns.GenerationRunsSnapshot: string(runsJSON),
		}).InsertIgnore()
		if insertErr != nil {
			return gerror.Wrap(insertErr, "failed to create demo baseline")
		}
		return nil
	})
}

// defaultDemoBaseCurrency is the base currency provisioned for a freshly created
// online demo user. It must exist in the currencies table (seeded by migrations).
const defaultDemoBaseCurrency = "USD"

// demoAccountSeeds is the chart of accounts provisioned for an online demo
// user. Every entry's name and type must satisfy at least one template in the
// transaction planner (see planner.go): a template whose source or destination
// cannot be matched is silently skipped, so a gap here produces zero generated
// transactions for the affected template.
var demoAccountSeeds = []struct {
	name        string
	accountType int
	isGroup     bool
}{
	{"Assets", int(utils.AccountTypeAsset), true},
	{"Checking", int(utils.AccountTypeAsset), false},
	{"Savings", int(utils.AccountTypeAsset), false},
	{"Auto Loan", int(utils.AccountTypeLiability), false},
	{"Interest", int(utils.AccountTypeIncome), false},
	{"Salary", int(utils.AccountTypeIncome), false},
	{"Rent", int(utils.AccountTypeExpense), false},
	{"Subscriptions", int(utils.AccountTypeExpense), false},
	{"Auto Insurance", int(utils.AccountTypeExpense), false},
	{"Groceries", int(utils.AccountTypeExpense), false},
	{"Dining", int(utils.AccountTypeExpense), false},
	{"Transport", int(utils.AccountTypeExpense), false},
	{"Shopping", int(utils.AccountTypeExpense), false},
	{"Entertainment", int(utils.AccountTypeExpense), false},
	{"Health & Fitness", int(utils.AccountTypeExpense), false},
	{"Opening Equity", int(utils.AccountTypeEquity), false},
}

// seedDemoAccounts idempotently provisions demoAccountSeeds for the user and
// wires the asset group, default child, and equity accounts.
func seedDemoAccounts(ctx context.Context, tx gdb.TX, user entity.Users) error {
	accountColumns := dao.Accounts.Columns()
	currency := strings.ToUpper(strings.TrimSpace(user.MainCurrency))
	if currency == "" {
		currency = defaultDemoBaseCurrency
	}

	var existing []entity.Accounts
	if err := tx.Model(dao.Accounts.Table()).
		Where(accountColumns.UserId, user.Id).
		WhereNull(accountColumns.DeletedAt).
		OrderAsc(accountColumns.Id).
		Scan(&existing); err != nil {
		return gerror.Wrap(err, "failed to load accounts for demo seeding")
	}
	idByName := make(map[string]uuid.UUID, len(existing)+len(demoAccountSeeds))
	for i := range existing {
		idByName[existing[i].Name] = existing[i].Id
	}

	for _, seed := range demoAccountSeeds {
		if _, exists := idByName[seed.name]; exists {
			continue
		}
		id := uuid.New()
		if _, err := tx.Model(dao.Accounts.Table()).Data(g.Map{
			accountColumns.Id:           id,
			accountColumns.UserId:       user.Id,
			accountColumns.Name:         seed.name,
			accountColumns.Type:         seed.accountType,
			accountColumns.IsGroup:      seed.isGroup,
			accountColumns.CurrencyCode: currency,
			accountColumns.BalanceUnits: int64(0),
			accountColumns.BalanceNanos: 0,
		}).Insert(); err != nil {
			return gerror.Wrapf(err, "failed to seed demo account %q", seed.name)
		}
		idByName[seed.name] = id
	}

	relationships := []struct {
		name    string
		payload g.Map
	}{
		{"Assets", g.Map{accountColumns.DefaultChildId: idByName["Checking"]}},
		{"Checking", g.Map{accountColumns.ParentId: idByName["Assets"], accountColumns.EquityAccountId: idByName["Opening Equity"]}},
		{"Savings", g.Map{accountColumns.ParentId: idByName["Assets"]}},
	}
	for _, ref := range relationships {
		id, exists := idByName[ref.name]
		if !exists {
			return gerror.Newf("demo account %q missing after seeding", ref.name)
		}
		if _, err := tx.Model(dao.Accounts.Table()).Where(accountColumns.Id, id).Data(ref.payload).Update(); err != nil {
			return gerror.Wrap(err, "failed to wire demo user account relationships")
		}
	}
	return nil
}

// createDemoUser provisions the configured online demo user when it is missing
// from the database so a fresh environment can start without manual seeding. The
// new account ships with the full seeded chart of accounts so the planner can
// generate income, expense, and transfer transactions immediately.
func (s *sDemoData) createDemoUser(ctx context.Context, config Config) (*entity.Users, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(config.UserPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to hash configured demo password")
	}
	userID := uuid.New()

	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		userColumns := dao.Users.Columns()
		if _, err := tx.Model(dao.Users.Table()).Data(g.Map{
			userColumns.Id:               userID,
			userColumns.Email:            config.UserEmail,
			userColumns.Nickname:         "Online Demo",
			userColumns.Avatar:           "",
			userColumns.Plan:             1,
			userColumns.Password:         string(hashedPassword),
			userColumns.MainCurrency:     defaultDemoBaseCurrency,
			userColumns.ThemeId:          nil,
			userColumns.TwoFactorSecret:  nil,
			userColumns.TwoFactorEnabled: false,
		}).Insert(); err != nil {
			return gerror.Wrap(err, "failed to create configured demo user")
		}
		return seedDemoAccounts(ctx, tx, entity.Users{Id: userID, MainCurrency: defaultDemoBaseCurrency})
	})
	if err != nil {
		return nil, err
	}

	userColumns := dao.Users.Columns()
	var created entity.Users
	if err := dao.Users.Ctx(ctx).Where(userColumns.Id, userID).Scan(&created); err != nil {
		return nil, gerror.Wrap(err, "failed to reload configured demo user")
	}
	return &created, nil
}

func loadBaselineRows(ctx context.Context, tx gdb.TX, userID uuid.UUID) ([]demoAccountSnapshot, []demoTransactionSnapshot, []entity.DemoDataGenerationRuns, error) {
	accountColumns := dao.Accounts.Columns()
	var accountEntities []entity.Accounts
	if err := tx.Model(dao.Accounts.Table()).
		Where(accountColumns.UserId, userID).
		OrderAsc(accountColumns.Id).
		Scan(&accountEntities); err != nil {
		return nil, nil, nil, gerror.Wrap(err, "failed to load demo baseline accounts")
	}
	accounts := make([]demoAccountSnapshot, 0, len(accountEntities))
	for _, account := range accountEntities {
		accounts = append(accounts, demoAccountSnapshot{
			ID: account.Id, ParentID: account.ParentId, Name: account.Name, Type: account.Type,
			IsGroup: account.IsGroup, CurrencyCode: account.CurrencyCode,
			BalanceUnits: account.BalanceUnits, BalanceNanos: account.BalanceNanos,
			DefaultChildID: account.DefaultChildId, EquityAccountID: account.EquityAccountId,
			Date: account.Date, Number: account.Number, Remarks: account.Remarks,
			CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt, DeletedAt: account.DeletedAt,
		})
	}

	transactionColumns := dao.Transactions.Columns()
	var transactionEntities []entity.Transactions
	if err := tx.Model(dao.Transactions.Table()).
		Where(transactionColumns.UserId, userID).
		OrderAsc(transactionColumns.Id).
		Scan(&transactionEntities); err != nil {
		return nil, nil, nil, gerror.Wrap(err, "failed to load demo baseline transactions")
	}
	transactions := make([]demoTransactionSnapshot, 0, len(transactionEntities))
	for _, transaction := range transactionEntities {
		transactions = append(transactions, demoTransactionSnapshot{
			ID: transaction.Id, Date: transaction.Date,
			FromAccountID: transaction.FromAccountId, ToAccountID: transaction.ToAccountId,
			CurrencyCode: transaction.CurrencyCode, BalanceUnits: transaction.BalanceUnits,
			BalanceNanos: transaction.BalanceNanos, Note: transaction.Note, Type: transaction.Type,
			CreatedAt: transaction.CreatedAt, UpdatedAt: transaction.UpdatedAt, DeletedAt: transaction.DeletedAt,
		})
	}

	runColumns := dao.DemoDataGenerationRuns.Columns()
	var runs []entity.DemoDataGenerationRuns
	if err := tx.Model(dao.DemoDataGenerationRuns.Table()).
		Where(runColumns.UserId, userID).
		OrderAsc(runColumns.BusinessDate).
		Scan(&runs); err != nil {
		return nil, nil, nil, gerror.Wrap(err, "failed to load demo generation baseline")
	}
	return accounts, transactions, runs, nil
}

func (s *sDemoData) resetForDate(ctx context.Context, config Config, resetDate time.Time) (bool, error) {
	user, err := loadDemoUser(ctx, config.UserEmail)
	if err != nil {
		return false, err
	}
	resetDateText := resetDate.In(config.Location).Format(time.DateOnly)
	cacheAccountIDs := make([]string, 0)
	resetPerformed := false

	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		baselineColumns := dao.DemoUserBaselines.Columns()
		var baseline entity.DemoUserBaselines
		if scanErr := tx.Model(dao.DemoUserBaselines.Table()).
			Where(baselineColumns.UserId, user.Id).
			LockUpdate().
			Scan(&baseline); scanErr != nil {
			return gerror.Wrap(scanErr, "failed to lock demo baseline")
		}
		if baseline.UserId == uuid.Nil {
			return gerror.New("demo baseline was not found")
		}
		if baseline.LastResetDate != nil && baseline.LastResetDate.Format("Y-m-d") == resetDateText {
			return nil
		}

		var userSnapshot demoUserSnapshot
		var accounts []demoAccountSnapshot
		var transactions []demoTransactionSnapshot
		var runs []entity.DemoDataGenerationRuns
		if decodeErr := json.Unmarshal([]byte(baseline.UserSnapshot), &userSnapshot); decodeErr != nil {
			return gerror.Wrap(decodeErr, "failed to decode demo user baseline")
		}
		if decodeErr := json.Unmarshal([]byte(baseline.AccountsSnapshot), &accounts); decodeErr != nil {
			return gerror.Wrap(decodeErr, "failed to decode demo account baseline")
		}
		if decodeErr := json.Unmarshal([]byte(baseline.TransactionsSnapshot), &transactions); decodeErr != nil {
			return gerror.Wrap(decodeErr, "failed to decode demo transaction baseline")
		}
		if decodeErr := json.Unmarshal([]byte(baseline.GenerationRunsSnapshot), &runs); decodeErr != nil {
			return gerror.Wrap(decodeErr, "failed to decode demo generation baseline")
		}

		accountColumns := dao.Accounts.Columns()
		var currentAccountIDs []string
		if scanErr := tx.Model(dao.Accounts.Table()).Fields(accountColumns.Id).
			Where(accountColumns.UserId, user.Id).Scan(&currentAccountIDs); scanErr != nil {
			return gerror.Wrap(scanErr, "failed to load current demo account cache keys")
		}
		cacheAccountIDs = append(cacheAccountIDs, currentAccountIDs...)
		for _, account := range accounts {
			cacheAccountIDs = append(cacheAccountIDs, account.ID.String())
		}

		if restoreErr := restoreBaseline(ctx, tx, *user, userSnapshot, accounts, transactions, runs); restoreErr != nil {
			return restoreErr
		}
		if _, updateErr := tx.Model(dao.DemoUserBaselines.Table()).
			Where(baselineColumns.UserId, user.Id).
			Data(g.Map{
				baselineColumns.LastResetDate: resetDateText,
				baselineColumns.UpdatedAt:     gtime.Now(),
			}).Update(); updateErr != nil {
			return gerror.Wrap(updateErr, "failed to mark demo baseline reset")
		}
		resetPerformed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	if !resetPerformed {
		return false, nil
	}

	keys := []string{utils.UserCacheKey(user.Id.String())}
	seen := make(map[string]struct{}, len(cacheAccountIDs))
	for _, accountID := range cacheAccountIDs {
		accountID = strings.TrimSpace(accountID)
		if accountID == "" {
			continue
		}
		if _, ok := seen[accountID]; ok {
			continue
		}
		seen[accountID] = struct{}{}
		keys = append(keys, utils.AccountCacheKey(accountID))
	}
	_ = utils.InvalidateCache(ctx, keys...)
	dashboard.PublishDashboardRefresh(ctx, user.Id.String(), "demo_data_reset")
	return true, nil
}

func restoreBaseline(ctx context.Context, tx gdb.TX, user entity.Users, userSnapshot demoUserSnapshot, accounts []demoAccountSnapshot, transactions []demoTransactionSnapshot, runs []entity.DemoDataGenerationRuns) error {
	dashboardColumns := dao.DashboardSnapshots.Columns()
	taskColumns := dao.Tasks.Columns()
	transactionColumns := dao.Transactions.Columns()
	accountColumns := dao.Accounts.Columns()
	runColumns := dao.DemoDataGenerationRuns.Columns()

	deleteTargets := []struct {
		table  string
		column string
	}{
		{dao.DashboardSnapshots.Table(), dashboardColumns.UserId},
		{dao.Tasks.Table(), taskColumns.UserId},
		{dao.Transactions.Table(), transactionColumns.UserId},
		{dao.Accounts.Table(), accountColumns.UserId},
		{dao.DemoDataGenerationRuns.Table(), runColumns.UserId},
	}
	for _, target := range deleteTargets {
		if _, err := tx.Model(target.table).Unscoped().Where(target.column, user.Id).Delete(); err != nil {
			return gerror.Wrap(err, "failed to clear mutable demo data")
		}
	}

	userColumns := dao.Users.Columns()
	userData := g.Map{
		userColumns.Password:         userSnapshot.Password,
		userColumns.Nickname:         userSnapshot.Nickname,
		userColumns.Avatar:           userSnapshot.Avatar,
		userColumns.Plan:             userSnapshot.Plan,
		userColumns.MainCurrency:     userSnapshot.MainCurrency,
		userColumns.TwoFactorSecret:  userSnapshot.TwoFactorSecret,
		userColumns.TwoFactorEnabled: userSnapshot.TwoFactorEnabled,
	}
	if userSnapshot.ThemeID == uuid.Nil {
		userData[userColumns.ThemeId] = nil
	} else {
		userData[userColumns.ThemeId] = userSnapshot.ThemeID
	}
	if _, err := tx.Model(dao.Users.Table()).Where(userColumns.Id, user.Id).Data(userData).Update(); err != nil {
		return gerror.Wrap(err, "failed to restore demo user")
	}

	for _, account := range accounts {
		data := g.Map{
			accountColumns.Id: account.ID, accountColumns.UserId: user.Id,
			accountColumns.Name: account.Name, accountColumns.Type: account.Type,
			accountColumns.IsGroup: account.IsGroup, accountColumns.CurrencyCode: account.CurrencyCode,
			accountColumns.BalanceUnits: account.BalanceUnits, accountColumns.BalanceNanos: account.BalanceNanos,
			accountColumns.Date: account.Date, accountColumns.Number: account.Number,
			accountColumns.Remarks: account.Remarks, accountColumns.CreatedAt: account.CreatedAt,
			accountColumns.UpdatedAt: account.UpdatedAt, accountColumns.DeletedAt: account.DeletedAt,
		}
		if _, err := tx.Model(dao.Accounts.Table()).Data(data).Insert(); err != nil {
			return gerror.Wrap(err, "failed to restore demo account")
		}
	}
	for _, account := range accounts {
		references := g.Map{}
		if account.ParentID != uuid.Nil {
			references[accountColumns.ParentId] = account.ParentID
		}
		if account.DefaultChildID != uuid.Nil {
			references[accountColumns.DefaultChildId] = account.DefaultChildID
		}
		if account.EquityAccountID != uuid.Nil {
			references[accountColumns.EquityAccountId] = account.EquityAccountID
		}
		if len(references) > 0 {
			if _, err := tx.Model(dao.Accounts.Table()).Where(accountColumns.Id, account.ID).Data(references).Update(); err != nil {
				return gerror.Wrap(err, "failed to restore demo account relationships")
			}
		}
	}

	for _, transaction := range transactions {
		data := g.Map{
			transactionColumns.Id: transaction.ID, transactionColumns.UserId: user.Id,
			transactionColumns.Date: transaction.Date, transactionColumns.FromAccountId: transaction.FromAccountID,
			transactionColumns.ToAccountId: transaction.ToAccountID, transactionColumns.CurrencyCode: transaction.CurrencyCode,
			transactionColumns.BalanceUnits: transaction.BalanceUnits, transactionColumns.BalanceNanos: transaction.BalanceNanos,
			transactionColumns.Note: transaction.Note, transactionColumns.Type: transaction.Type,
			transactionColumns.CreatedAt: transaction.CreatedAt, transactionColumns.UpdatedAt: transaction.UpdatedAt,
			transactionColumns.DeletedAt: transaction.DeletedAt,
		}
		if _, err := tx.Model(dao.Transactions.Table()).Data(data).Insert(); err != nil {
			return gerror.Wrap(err, "failed to restore demo transaction")
		}
	}
	for _, run := range runs {
		if _, err := tx.Model(dao.DemoDataGenerationRuns.Table()).Data(g.Map{
			runColumns.UserId: user.Id, runColumns.BusinessDate: run.BusinessDate,
			runColumns.GeneratedCount: run.GeneratedCount, runColumns.CreatedAt: run.CreatedAt,
		}).Insert(); err != nil {
			return gerror.Wrap(err, "failed to restore demo generation run")
		}
	}
	return nil
}
