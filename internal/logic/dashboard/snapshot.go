package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/mq"
	internalRedis "gaap-api/internal/redis"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

// ─── Redis Key Helpers ───────────────────────────────────────────────────────

const (
	// Snapshot keys (long-lived, updated on every data mutation)
	snapshotSummaryKey = "dashboard:snapshot:summary:%s"
	snapshotMonthlyKey = "dashboard:snapshot:monthly:%s:%s" // userId + YYYY-MM
	snapshotTrendKey   = "dashboard:snapshot:trend:%s"      // userId

	// Snapshot TTL — long lived; refreshed on every mutation event
	snapshotTTL = 24 * time.Hour
)

func summarySnapshotKey(userId string) string {
	return fmt.Sprintf(snapshotSummaryKey, userId)
}

func monthlySnapshotKey(userId string) string {
	month := time.Now().Format("2006-01")
	return fmt.Sprintf(snapshotMonthlyKey, userId, month)
}

func trendSnapshotKey(userId string) string {
	return fmt.Sprintf(snapshotTrendKey, userId)
}

// ─── MQ Message Types ────────────────────────────────────────────────────────

// MQ message type for dashboard refresh events
const (
	MsgTypeDashboardRefresh = 100 // distinct from task types (1-3)
)

// DashboardRefreshPayload is the MQ message payload for dashboard refresh
type DashboardRefreshPayload struct {
	UserId string `json:"userId"`
	Reason string `json:"reason"` // "tx_create", "tx_update", "tx_delete", "account_update", "full_rebuild"
}

// ─── Public: Publish Refresh Event ───────────────────────────────────────────

// PublishDashboardRefresh enqueues a dashboard snapshot refresh via RabbitMQ.
// Called after any transaction or account balance mutation.
// It's fire-and-forget — dashboard still serves stale snapshot on failure.
func PublishDashboardRefresh(ctx context.Context, userId string, reason string) {
	// Remove both cache tiers before publishing. An immediate dashboard read
	// must not restore a stale persisted snapshot while the worker is debouncing.
	invalidateDashboardSnapshots(ctx, userId)

	client := mq.GetRabbitMQ()
	if !client.IsConnected() {
		return
	}

	payload := DashboardRefreshPayload{
		UserId: userId,
		Reason: reason,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to marshal dashboard refresh payload: %v", err)
		return
	}

	msg := &mq.Message{
		Type:    MsgTypeDashboardRefresh,
		Payload: payloadBytes,
	}

	if err := client.Publish(ctx, mq.QueueDashboard, msg); err != nil {
		g.Log().Warningf(ctx, "Failed to publish dashboard refresh (will recalculate on next request): %v", err)
	}
}

func invalidateDashboardSnapshots(ctx context.Context, userId string) {
	_ = utils.InvalidateCache(ctx,
		summarySnapshotKey(userId),
		monthlySnapshotKey(userId),
		trendSnapshotKey(userId),
	)
	if _, err := dao.DashboardSnapshots.Ctx(ctx).
		Where(dao.DashboardSnapshots.Columns().UserId, userId).
		Delete(); err != nil {
		g.Log().Warningf(ctx, "Failed to invalidate persisted dashboard snapshots for user %s: %v", userId, err)
	}
}

// ─── Public: Snapshot Read (O(1) from Redis) ─────────────────────────────────

// GetSummarySnapshot reads the dashboard summary from Redis snapshot.
// Falls back to DB computation + snapshot write on cache miss.
func GetSummarySnapshot(ctx context.Context, userId string) (*model.DashboardSummary, error) {
	return getOrBuildSnapshot(
		ctx,
		summarySnapshotKey(userId),
		&snapshotMeta{userId: userId, snapshotType: SnapshotTypeSummary, snapshotKey: ""},
		func(ctx context.Context) (*model.DashboardSummary, error) {
			svc := New()
			return svc.loadDashboardSummaryFromDB(ctx, userId)
		},
	)
}

// GetMonthlySnapshot reads the monthly stats from Redis snapshot.
func GetMonthlySnapshot(ctx context.Context, userId string) (*model.MonthlyStats, error) {
	monthKey := time.Now().Format("2006-01")
	return getOrBuildSnapshot(
		ctx,
		monthlySnapshotKey(userId),
		&snapshotMeta{userId: userId, snapshotType: SnapshotTypeMonthly, snapshotKey: monthKey},
		func(ctx context.Context) (*model.MonthlyStats, error) {
			svc := New()
			return svc.loadMonthlyStatsFromDB(ctx, userId)
		},
	)
}

// GetTrendSnapshot reads the balance trend from Redis snapshot.
func GetTrendSnapshot(ctx context.Context, userId string, accounts []uuid.UUID) ([]model.DailyBalance, error) {
	key := trendSnapshotKey(userId)

	client, err := internalRedis.GetCacheClient(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "Redis unavailable for trend snapshot, trying DB: %v", err)
		// Try DB
		dbResult, dbErr := LoadSnapshotFromDB[[]model.DailyBalance](ctx, userId, SnapshotTypeTrend, "")
		if dbErr == nil && dbResult != nil {
			return *dbResult, nil
		}
		svc := New()
		return svc.loadBalanceTrendFromDB(ctx, userId, accounts)
	}

	// 1st: Redis
	cached, err := client.Get(ctx, key)
	if err == nil && !cached.IsNil() {
		var result []model.DailyBalance
		if jsonErr := json.Unmarshal(cached.Bytes(), &result); jsonErr == nil {
			return result, nil
		}
	}

	// 2nd: DB
	dbResult, dbErr := LoadSnapshotFromDB[[]model.DailyBalance](ctx, userId, SnapshotTypeTrend, "")
	if dbErr == nil && dbResult != nil {
		g.Log().Debugf(ctx, "Trend snapshot HIT (DB) for user %s", userId)
		go writeTrendSnapshot(client, key, *dbResult)
		return *dbResult, nil
	}

	// 3rd: Full recompute
	svc := New()
	result, err := svc.loadBalanceTrendFromDB(ctx, userId, accounts)
	if err != nil {
		return nil, err
	}

	go writeTrendSnapshot(client, key, result)
	return result, nil
}

// ─── Public: Snapshot Rebuild (called by MQ worker) ──────────────────────────

// RebuildSnapshots recomputes all dashboard snapshots for a user and writes to Redis.
// This is the heavy-lifting function called by the MQ consumer.
func RebuildSnapshots(ctx context.Context, userId string) error {
	g.Log().Infof(ctx, "Rebuilding dashboard snapshots for user %s", userId)

	svc := New()

	// 1. Rebuild summary
	summary, err := svc.loadDashboardSummaryFromDB(ctx, userId)
	if err != nil {
		return gerror.Wrapf(err, "failed to rebuild summary snapshot for user %s", userId)
	}
	if err := writeSnapshot(ctx, summarySnapshotKey(userId), summary); err != nil {
		g.Log().Errorf(ctx, "Failed to write summary snapshot: %v", err)
	}

	// 2. Rebuild monthly stats
	monthly, err := svc.loadMonthlyStatsFromDB(ctx, userId)
	if err != nil {
		return gerror.Wrapf(err, "failed to rebuild monthly snapshot for user %s", userId)
	}
	if err := writeSnapshot(ctx, monthlySnapshotKey(userId), monthly); err != nil {
		g.Log().Errorf(ctx, "Failed to write monthly snapshot: %v", err)
	}

	// 3. Rebuild balance trend (all accounts)
	trend, err := svc.loadBalanceTrendFromDB(ctx, userId, nil)
	if err != nil {
		return gerror.Wrapf(err, "failed to rebuild trend snapshot for user %s", userId)
	}
	if err := writeSnapshot(ctx, trendSnapshotKey(userId), trend); err != nil {
		g.Log().Errorf(ctx, "Failed to write trend snapshot: %v", err)
	}

	// Also invalidate the legacy short-TTL cache keys
	_ = utils.InvalidateCache(ctx,
		utils.DashboardSummaryCacheKey(userId),
		utils.DashboardMonthlyCacheKey(userId),
	)

	// Persist snapshots to DB for durability
	go func() {
		monthKey := time.Now().Format("2006-01")
		if err := PersistSnapshotToDB(context.Background(), userId, SnapshotTypeSummary, "", summary); err != nil {
			g.Log().Warningf(context.Background(), "Failed to persist summary to DB: %v", err)
		}
		if err := PersistSnapshotToDB(context.Background(), userId, SnapshotTypeMonthly, monthKey, monthly); err != nil {
			g.Log().Warningf(context.Background(), "Failed to persist monthly to DB: %v", err)
		}
		if err := PersistSnapshotToDB(context.Background(), userId, SnapshotTypeTrend, "", trend); err != nil {
			g.Log().Warningf(context.Background(), "Failed to persist trend to DB: %v", err)
		}
	}()

	g.Log().Infof(ctx, "Dashboard snapshots rebuilt successfully for user %s", userId)
	return nil
}

// ─── Internal: Generic snapshot read/write ───────────────────────────────────

// snapshotMeta holds the DB coordinates for a snapshot so the generic function
// can fall back to DB when Redis is empty.
type snapshotMeta struct {
	userId       string
	snapshotType string
	snapshotKey  string
}

func getOrBuildSnapshot[T any](
	ctx context.Context,
	key string,
	meta *snapshotMeta,
	loader func(ctx context.Context) (*T, error),
) (*T, error) {
	client, err := internalRedis.GetCacheClient(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "Redis unavailable for snapshot, trying DB fallback: %v", err)
		return loadFromDBOrCompute(ctx, meta, loader)
	}

	// 1st: Try read from Redis snapshot
	cached, err := client.Get(ctx, key)
	if err == nil && !cached.IsNil() {
		var result T
		if jsonErr := json.Unmarshal(cached.Bytes(), &result); jsonErr == nil {
			g.Log().Debug(ctx, "Dashboard snapshot cache hit")
			return &result, nil
		}
		g.Log().Warning(ctx, "Dashboard snapshot cache decode failed, trying database")
	}

	// 2nd: Try read from DB
	if meta != nil {
		dbResult, dbErr := LoadSnapshotFromDB[T](ctx, meta.userId, meta.snapshotType, meta.snapshotKey)
		if dbErr == nil && dbResult != nil {
			g.Log().Debugf(ctx, "Snapshot HIT (DB): %s/%s", meta.snapshotType, meta.snapshotKey)
			// Restore to Redis asynchronously
			go func() {
				data, _ := json.Marshal(dbResult)
				if data != nil {
					_ = client.SetEX(context.Background(), key, data, int64(snapshotTTL.Seconds()))
				}
			}()
			return dbResult, nil
		}
	}

	// 3rd: Full recompute from transactional data
	g.Log().Debug(ctx, "Dashboard snapshot cache miss, computing from source")
	result, err := loader(ctx)
	if err != nil {
		return nil, err
	}

	// Write to Redis asynchronously
	go func() {
		data, _ := json.Marshal(result)
		if data != nil {
			_ = client.SetEX(context.Background(), key, data, int64(snapshotTTL.Seconds()))
		}
	}()

	return result, nil
}

// loadFromDBOrCompute tries DB first, falls back to full computation.
func loadFromDBOrCompute[T any](
	ctx context.Context,
	meta *snapshotMeta,
	loader func(ctx context.Context) (*T, error),
) (*T, error) {
	if meta != nil {
		dbResult, dbErr := LoadSnapshotFromDB[T](ctx, meta.userId, meta.snapshotType, meta.snapshotKey)
		if dbErr == nil && dbResult != nil {
			g.Log().Debugf(ctx, "Snapshot HIT (DB, no Redis): %s/%s", meta.snapshotType, meta.snapshotKey)
			return dbResult, nil
		}
	}
	return loader(ctx)
}

func writeSnapshot(ctx context.Context, key string, value interface{}) error {
	client, err := internalRedis.GetCacheClient(ctx)
	if err != nil {
		return gerror.Wrap(err, "redis unavailable for snapshot write")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return gerror.Wrap(err, "failed to marshal snapshot data")
	}

	return client.SetEX(ctx, key, data, int64(snapshotTTL.Seconds()))
}

func writeTrendSnapshot(client *gredis.Redis, key string, data []model.DailyBalance) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = client.SetEX(context.Background(), key, bytes, int64(snapshotTTL.Seconds()))
}

// ─── loadBalanceTrendFromDB extracted from original GetBalanceTrend ───────────

type accountBalance struct {
	Id           uuid.UUID
	BalanceUnits int64
	BalanceNanos int
	CurrencyCode string
}

func (s *sDashboard) loadBalanceTrendFromDB(ctx context.Context, userId string, accounts []uuid.UUID) ([]model.DailyBalance, error) {
	now := time.Now()
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	startDate := endDate.AddDate(0, 0, -29)
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	// If no specific accounts requested, get all user's non-group accounts
	if len(accounts) == 0 {
		var userAccounts []entity.Accounts
		err := dao.Accounts.Ctx(ctx).
			Where(dao.Accounts.Columns().UserId, userId).
			Where(dao.Accounts.Columns().IsGroup, false).
			WhereNull(dao.Accounts.Columns().DeletedAt).
			Fields(dao.Accounts.Columns().Id).
			Scan(&userAccounts)
		if err != nil {
			return nil, gerror.Wrap(err, "failed to get user accounts")
		}
		for _, acc := range userAccounts {
			accounts = append(accounts, acc.Id)
		}
	}

	if len(accounts) == 0 {
		return []model.DailyBalance{}, nil
	}

	// 1. Get CURRENT balances for these accounts
	currentBalances := make(map[uuid.UUID]accountBalance)
	var accountRecs []entity.Accounts
	err := dao.Accounts.Ctx(ctx).
		WhereIn(dao.Accounts.Columns().Id, accounts).
		Where(dao.Accounts.Columns().UserId, userId).
		Scan(&accountRecs)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get account balances")
	}
	for _, acc := range accountRecs {
		currentBalances[acc.Id] = accountBalance{
			Id:           acc.Id,
			BalanceUnits: acc.BalanceUnits,
			BalanceNanos: acc.BalanceNanos,
			CurrencyCode: acc.CurrencyCode,
		}
	}

	// 2. Get ALL transactions for these accounts from startDate to NOW
	var fromTrans []entity.Transactions
	var toTrans []entity.Transactions

	err = dao.Transactions.Ctx(ctx).
		WhereIn(dao.Transactions.Columns().FromAccountId, accounts).
		WhereGTE(dao.Transactions.Columns().Date, startOfDay).
		WhereNull(dao.Transactions.Columns().DeletedAt).
		Limit(10000).
		Scan(&fromTrans)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get from transactions")
	}

	err = dao.Transactions.Ctx(ctx).
		WhereIn(dao.Transactions.Columns().ToAccountId, accounts).
		WhereGTE(dao.Transactions.Columns().Date, startOfDay).
		WhereNull(dao.Transactions.Columns().DeletedAt).
		Limit(10000).
		Scan(&toTrans)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get to transactions")
	}

	// Merge and deduplicate transactions
	txMap := make(map[uuid.UUID]entity.Transactions)
	for _, t := range fromTrans {
		txMap[t.Id] = t
	}
	for _, t := range toTrans {
		txMap[t.Id] = t
	}

	transactions := make([]entity.Transactions, 0, len(txMap))
	for _, t := range txMap {
		transactions = append(transactions, t)
	}

	return calculateBalanceTrend(now, currentBalances, transactions), nil
}

func calculateBalanceTrend(now time.Time, currentBalances map[uuid.UUID]accountBalance, transactions []entity.Transactions) []model.DailyBalance {
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	startDate := endDate.AddDate(0, 0, -29)
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	// Create a map of Date -> Transactions
	transactionsByDate := make(map[string][]entity.Transactions)
	for _, t := range transactions {
		if t.Date == nil {
			continue
		}
		dateStr := t.Date.Layout("2006-01-02")
		transactionsByDate[dateStr] = append(transactionsByDate[dateStr], t)
	}

	// 3. Calculate daily balances BACKWARDS using MoneyHelper
	runningBalances := make(map[uuid.UUID]*utils.MoneyHelper)
	for accId, bal := range currentBalances {
		accEntity := &entity.Accounts{
			BalanceUnits: bal.BalanceUnits,
			BalanceNanos: bal.BalanceNanos,
			CurrencyCode: bal.CurrencyCode,
		}
		runningBalances[accId] = utils.NewFromEntity(accEntity)
	}

	dailyMap := make(map[string]map[string]model.DailyAccountBalance)

	cursorDate := endDate
	for !cursorDate.Before(startOfDay) {
		dateStr := cursorDate.Format("2006-01-02")

		// Record the balance at the END of this day
		dayBalances := make(map[string]model.DailyAccountBalance)
		for accId, bal := range runningBalances {
			units, nanos := bal.ToEntityValues()
			dayBalances[accId.String()] = model.DailyAccountBalance{
				Units:        units,
				Nanos:        nanos,
				CurrencyCode: bal.Currency,
			}
		}
		dailyMap[dateStr] = dayBalances

		// Reverse transactions of this day to get start-of-day balances
		if txs, ok := transactionsByDate[dateStr]; ok {
			for _, tx := range txs {
				// Create delta MoneyHelper
				deltaEntity := &entity.Accounts{
					BalanceUnits: tx.BalanceUnits,
					BalanceNanos: tx.BalanceNanos,
					CurrencyCode: tx.CurrencyCode,
				}
				delta := utils.NewFromEntity(deltaEntity)

				// FromAccount: Money left, so add back
				if bal, ok := runningBalances[tx.FromAccountId]; ok {
					newBal, _ := bal.Add(delta)
					if newBal != nil {
						runningBalances[tx.FromAccountId] = newBal
					}
				}
				// ToAccount: Money entered, so subtract
				if bal, ok := runningBalances[tx.ToAccountId]; ok {
					newBal, _ := bal.Sub(delta)
					if newBal != nil {
						runningBalances[tx.ToAccountId] = newBal
					}
				}
			}
		}

		cursorDate = cursorDate.AddDate(0, 0, -1)
	}

	// 4. Construct final output (sorted by date)
	out := make([]model.DailyBalance, 0)
	for d := startOfDay; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if bals, ok := dailyMap[dateStr]; ok {
			out = append(out, model.DailyBalance{
				Date:     dateStr,
				Balances: bals,
			})
		} else {
			out = append(out, model.DailyBalance{
				Date:     dateStr,
				Balances: make(map[string]model.DailyAccountBalance),
			})
		}
	}

	return out
}
