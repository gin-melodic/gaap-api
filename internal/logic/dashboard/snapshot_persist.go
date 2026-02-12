package dashboard

import (
	"context"
	"encoding/json"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/model/entity"
	internalRedis "gaap-api/internal/redis"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// ─── Snapshot Type Constants ─────────────────────────────────────────────────

const (
	SnapshotTypeSummary = "summary"
	SnapshotTypeMonthly = "monthly"
	SnapshotTypeTrend   = "trend"
)

// ─── DB Persistence: Write ──────────────────────────────────────────────────

// PersistSnapshotToDB writes a single snapshot to the database via UPSERT.
// The (user_id, snapshot_type, snapshot_key) tuple is unique — existing rows are overwritten.
func PersistSnapshotToDB(ctx context.Context, userId string, snapshotType string, snapshotKey string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return gerror.Wrap(err, "failed to marshal snapshot for DB persistence")
	}

	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return gerror.Wrap(err, "invalid user ID for snapshot persistence")
	}

	// Try update first (faster path for existing snapshots)
	result, err := dao.DashboardSnapshots.Ctx(ctx).
		Where(dao.DashboardSnapshots.Columns().UserId, userUUID).
		Where(dao.DashboardSnapshots.Columns().SnapshotType, snapshotType).
		Where(dao.DashboardSnapshots.Columns().SnapshotKey, snapshotKey).
		Data(g.Map{
			dao.DashboardSnapshots.Columns().Data:      string(jsonBytes),
			dao.DashboardSnapshots.Columns().UpdatedAt: gtime.Now(),
		}).
		Update()
	if err != nil {
		return gerror.Wrap(err, "failed to update snapshot in DB")
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		return nil // Updated existing row
	}

	// No existing row — insert new
	newId, err := uuid.NewV7()
	if err != nil {
		return gerror.Wrap(err, "failed to generate UUID for snapshot")
	}

	_, err = dao.DashboardSnapshots.Ctx(ctx).Data(entity.DashboardSnapshots{
		Id:           newId,
		UserId:       userUUID,
		SnapshotType: snapshotType,
		SnapshotKey:  snapshotKey,
		Data:         string(jsonBytes),
	}).Insert()
	if err != nil {
		return gerror.Wrapf(err, "failed to insert snapshot [%s/%s] for user %s", snapshotType, snapshotKey, userId)
	}

	return nil
}

// PersistAllSnapshotsForUser flushes all 3 snapshot types from Redis → DB for one user.
func PersistAllSnapshotsForUser(ctx context.Context, userId string) error {
	g.Log().Debugf(ctx, "Persisting dashboard snapshots to DB for user %s", userId)

	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// 1. Summary
	summary, err := readSnapshotFromRedis[interface{}](ctx, summarySnapshotKey(userId))
	if err == nil && summary != nil {
		record(PersistSnapshotToDB(ctx, userId, SnapshotTypeSummary, "", summary))
	}

	// 2. Monthly (current month)
	monthly, err := readSnapshotFromRedis[interface{}](ctx, monthlySnapshotKey(userId))
	if err == nil && monthly != nil {
		monthKey := time.Now().Format("2006-01")
		record(PersistSnapshotToDB(ctx, userId, SnapshotTypeMonthly, monthKey, monthly))
	}

	// 3. Trend
	trend, err := readSnapshotFromRedis[interface{}](ctx, trendSnapshotKey(userId))
	if err == nil && trend != nil {
		record(PersistSnapshotToDB(ctx, userId, SnapshotTypeTrend, "", trend))
	}

	if firstErr != nil {
		g.Log().Warningf(ctx, "Partial failure persisting snapshots for user %s: %v", userId, firstErr)
	} else {
		g.Log().Debugf(ctx, "Snapshots persisted to DB successfully for user %s", userId)
	}
	return firstErr
}

// ─── DB Persistence: Read ───────────────────────────────────────────────────

// LoadSnapshotFromDB reads a snapshot from the database and deserializes it.
func LoadSnapshotFromDB[T any](ctx context.Context, userId string, snapshotType string, snapshotKey string) (*T, error) {
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return nil, gerror.Wrap(err, "invalid user ID")
	}

	var row entity.DashboardSnapshots
	err = dao.DashboardSnapshots.Ctx(ctx).
		Where(dao.DashboardSnapshots.Columns().UserId, userUUID).
		Where(dao.DashboardSnapshots.Columns().SnapshotType, snapshotType).
		Where(dao.DashboardSnapshots.Columns().SnapshotKey, snapshotKey).
		Scan(&row)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to query snapshot from DB")
	}

	if row.Id == uuid.Nil || row.Data == "" {
		return nil, nil // Not found
	}

	var result T
	if err := json.Unmarshal([]byte(row.Data), &result); err != nil {
		return nil, gerror.Wrapf(err, "failed to unmarshal snapshot [%s/%s] from DB", snapshotType, snapshotKey)
	}

	return &result, nil
}

// RestoreSnapshotsFromDB loads all persisted snapshots for a user from DB → Redis.
// Used on cold start to avoid full recomputation.
// Returns true if at least one snapshot was restored.
func RestoreSnapshotsFromDB(ctx context.Context, userId string) bool {
	restored := 0

	// 1. Summary
	summaryData, err := loadRawSnapshotFromDB(ctx, userId, SnapshotTypeSummary, "")
	if err == nil && summaryData != nil {
		if writeErr := writeSnapshot(ctx, summarySnapshotKey(userId), summaryData); writeErr == nil {
			restored++
		}
	}

	// 2. Monthly (current month)
	monthKey := time.Now().Format("2006-01")
	monthlyData, err := loadRawSnapshotFromDB(ctx, userId, SnapshotTypeMonthly, monthKey)
	if err == nil && monthlyData != nil {
		if writeErr := writeSnapshot(ctx, monthlySnapshotKey(userId), monthlyData); writeErr == nil {
			restored++
		}
	}

	// 3. Trend
	trendData, err := loadRawSnapshotFromDB(ctx, userId, SnapshotTypeTrend, "")
	if err == nil && trendData != nil {
		if writeErr := writeSnapshot(ctx, trendSnapshotKey(userId), trendData); writeErr == nil {
			restored++
		}
	}

	if restored > 0 {
		g.Log().Infof(ctx, "Restored %d snapshot(s) from DB for user %s", restored, userId)
	}
	return restored > 0
}

// loadRawSnapshotFromDB returns the raw JSON data as a json.RawMessage to avoid
// double-serialization when writing back to Redis.
func loadRawSnapshotFromDB(ctx context.Context, userId string, snapshotType string, snapshotKey string) (json.RawMessage, error) {
	userUUID, err := uuid.Parse(userId)
	if err != nil {
		return nil, err
	}

	var row entity.DashboardSnapshots
	err = dao.DashboardSnapshots.Ctx(ctx).
		Where(dao.DashboardSnapshots.Columns().UserId, userUUID).
		Where(dao.DashboardSnapshots.Columns().SnapshotType, snapshotType).
		Where(dao.DashboardSnapshots.Columns().SnapshotKey, snapshotKey).
		Scan(&row)
	if err != nil {
		return nil, err
	}

	if row.Id == uuid.Nil || row.Data == "" {
		return nil, nil
	}

	return json.RawMessage(row.Data), nil
}

// ─── Helper: Read raw snapshot from Redis ────────────────────────────────────

// readSnapshotFromRedis reads and deserialises a snapshot from Redis.
func readSnapshotFromRedis[T any](ctx context.Context, key string) (*T, error) {
	client, err := internalRedis.GetCacheClient(ctx)
	if err != nil {
		return nil, err
	}

	cached, err := client.Get(ctx, key)
	if err != nil || cached.IsNil() {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(cached.Bytes(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ─── Bulk Flush: All Users ──────────────────────────────────────────────────

// FlushAllSnapshotsToDB persists snapshots for ALL users from Redis → DB.
// Called by the periodic flush ticker.
func FlushAllSnapshotsToDB(ctx context.Context) {
	g.Log().Info(ctx, "Starting periodic snapshot flush to DB...")

	var users []struct {
		Id string `orm:"id"`
	}
	err := g.DB().Model("users").Fields("id").Scan(&users)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to query users for snapshot flush: %v", err)
		return
	}

	flushed := 0
	for _, u := range users {
		if err := PersistAllSnapshotsForUser(ctx, u.Id); err == nil {
			flushed++
		}
	}

	g.Log().Infof(ctx, "Snapshot flush completed: %d/%d users persisted", flushed, len(users))
}
