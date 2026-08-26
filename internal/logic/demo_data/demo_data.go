package demo_data

import (
	"context"
	"strings"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

const retryInterval = 15 * time.Minute

type sDemoData struct{}

func init() {
	service.RegisterDemoData(New())
}

func New() *sDemoData {
	return &sDemoData{}
}

// StartScheduler validates the online demo, captures its immutable baseline,
// and starts the daily reset/catch-up loop.
func (s *sDemoData) StartScheduler(ctx context.Context) error {
	config, err := LoadConfig(ctx)
	if err != nil {
		return err
	}
	if !config.OnlineDemoEnabled {
		g.Log().Info(ctx, "Online demo is disabled")
		return nil
	}
	if err := s.ensureBaseline(ctx, config); err != nil {
		return err
	}

	go s.runScheduler(ctx, config)
	return nil
}

// CatchUp generates every unfinished business date from the configured start
// date through yesterday in the configured timezone.
func (s *sDemoData) CatchUp(ctx context.Context) (int, error) {
	config, err := LoadConfig(ctx)
	if err != nil {
		return 0, err
	}
	if !config.Enabled {
		return 0, nil
	}
	return s.catchUpWithConfig(ctx, config, time.Now())
}

func (s *sDemoData) runScheduler(ctx context.Context, config Config) {
	for {
		now := time.Now()
		_, err := s.resetForDate(ctx, config, now)
		if err == nil && config.Enabled {
			_, err = s.catchUpWithConfig(ctx, config, now)
		}
		wait := durationUntilNextMidnight(time.Now(), config.Location)
		if err != nil {
			g.Log().Errorf(ctx, "Online demo daily cycle failed: %v", err)
			wait = retryInterval
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (s *sDemoData) catchUpWithConfig(ctx context.Context, config Config, now time.Time) (int, error) {
	user, err := loadDemoUser(ctx, config.UserEmail)
	if err != nil {
		return 0, err
	}

	endDate := catchUpEndDate(now, config)
	totalGenerated := 0
	processedDays := 0
	for date := config.StartDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		generated, alreadyProcessed, generateErr := s.generateBusinessDate(ctx, *user, date, config.Location)
		if generateErr != nil {
			return totalGenerated, gerror.Wrapf(generateErr, "failed to generate demo data for %s", date.Format(time.DateOnly))
		}
		if !alreadyProcessed {
			processedDays++
			totalGenerated += generated
		}
	}

	if totalGenerated > 0 {
		invalidateDemoUserCaches(ctx, user.Id.String())
		dashboard.PublishDashboardRefresh(ctx, user.Id.String(), "demo_data_catch_up")
	}
	if processedDays > 0 {
		g.Log().Infof(ctx, "Demo data catch-up completed: days=%d transactions=%d through=%s", processedDays, totalGenerated, endDate.Format(time.DateOnly))
	}
	return totalGenerated, nil
}

func (s *sDemoData) generateBusinessDate(
	ctx context.Context,
	user entity.Users,
	businessDate time.Time,
	location *time.Location,
) (generated int, alreadyProcessed bool, err error) {
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		runColumns := dao.DemoDataGenerationRuns.Columns()
		result, insertErr := tx.Model(dao.DemoDataGenerationRuns.Table()).Data(g.Map{
			runColumns.UserId:         user.Id,
			runColumns.BusinessDate:   businessDate.Format(time.DateOnly),
			runColumns.GeneratedCount: 0,
		}).InsertIgnore()
		if insertErr != nil {
			return gerror.Wrap(insertErr, "failed to reserve demo generation date")
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return gerror.Wrap(rowsErr, "failed to inspect demo generation reservation")
		}
		if rows == 0 {
			alreadyProcessed = true
			return nil
		}

		accounts, loadErr := lockDemoAccounts(ctx, tx, user)
		if loadErr != nil {
			return loadErr
		}
		planned, planErr := planTransactions(user, accounts, businessDate, location)
		if planErr != nil {
			return gerror.Wrap(planErr, "failed to plan demo transactions")
		}
		for _, transaction := range planned {
			if _, createErr := service.Transaction().CreateTransaction(ctx, transaction.input, tx); createErr != nil {
				return gerror.Wrap(createErr, "failed to create demo transaction")
			}
		}

		generated = len(planned)
		updateResult, updateErr := tx.Model(dao.DemoDataGenerationRuns.Table()).
			Where(runColumns.UserId, user.Id).
			Where(runColumns.BusinessDate, businessDate.Format(time.DateOnly)).
			Data(g.Map{runColumns.GeneratedCount: generated}).
			Update()
		if updateErr != nil {
			return gerror.Wrap(updateErr, "failed to complete demo generation date")
		}
		updatedRows, updatedRowsErr := updateResult.RowsAffected()
		if updatedRowsErr != nil || updatedRows != 1 {
			return gerror.New("demo generation completion did not affect exactly one row")
		}
		return nil
	})
	return generated, alreadyProcessed, err
}

func loadDemoUser(ctx context.Context, email string) (*entity.Users, error) {
	columns := dao.Users.Columns()
	var user entity.Users
	err := dao.Users.Ctx(ctx).
		Fields(columns.Id, columns.Email, columns.MainCurrency).
		Where(columns.Email, strings.ToLower(strings.TrimSpace(email))).
		WhereNull(columns.DeletedAt).
		Scan(&user)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to load demo user")
	}
	if user.Id.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, gerror.New("configured demo user was not found")
	}
	if strings.TrimSpace(user.MainCurrency) == "" {
		return nil, gerror.New("configured demo user has no base currency")
	}
	return &user, nil
}

func lockDemoAccounts(ctx context.Context, tx gdb.TX, user entity.Users) ([]entity.Accounts, error) {
	columns := dao.Accounts.Columns()
	var accounts []entity.Accounts
	err := tx.Model(dao.Accounts.Table()).
		Where(columns.UserId, user.Id).
		Where(columns.IsGroup, false).
		Where(columns.CurrencyCode, strings.ToUpper(user.MainCurrency)).
		WhereNull(columns.DeletedAt).
		OrderAsc(columns.Id).
		LockUpdate().
		Scan(&accounts)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to lock demo accounts")
	}
	return accounts, nil
}

func invalidateDemoUserCaches(ctx context.Context, userID string) {
	columns := dao.Accounts.Columns()
	var accountIDs []string
	if err := dao.Accounts.Ctx(ctx).
		Fields(columns.Id).
		Where(columns.UserId, userID).
		WhereNull(columns.DeletedAt).
		Scan(&accountIDs); err != nil {
		g.Log().Warningf(ctx, "Failed to load demo account cache keys: %v", err)
		return
	}
	keys := make([]string, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		keys = append(keys, utils.AccountCacheKey(accountID))
	}
	if len(keys) > 0 {
		_ = utils.InvalidateCache(ctx, keys...)
	}
}

func durationUntilNextMidnight(now time.Time, location *time.Location) time.Duration {
	localNow := now.In(location)
	nextMidnight := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
	return nextMidnight.Sub(localNow)
}

func catchUpEndDate(now time.Time, config Config) time.Time {
	today := now.In(config.Location)
	yesterday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, config.Location).AddDate(0, 0, -1)
	if yesterday.Before(config.InitialBackfillEndDate) {
		return config.InitialBackfillEndDate
	}
	return yesterday
}
