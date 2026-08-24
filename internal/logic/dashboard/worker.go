package dashboard

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/mq"

	"github.com/gogf/gf/v2/frame/g"
)

// debounceInterval prevents rebuilding snapshots more than once per user within this window.
// Multiple rapid mutations (e.g. batch import) will coalesce into a single rebuild.
const debounceInterval = 2 * time.Second

// pending tracks in-flight debounce timers per user
var (
	pending   = make(map[string]*time.Timer)
	pendingMu sync.Mutex
)

// StartDashboardWorker starts consuming dashboard refresh events from RabbitMQ.
// Should be called during application bootstrap (boot.go).
func StartDashboardWorker(ctx context.Context) error {
	g.Log().Info(ctx, "Starting dashboard snapshot worker...")

	return mq.GetRabbitMQ().Consume(ctx, mq.QueueDashboard, func(ctx context.Context, msg *mq.Message) error {
		if msg.Type != MsgTypeDashboardRefresh {
			g.Log().Warningf(ctx, "Unknown dashboard message type: %d", msg.Type)
			return nil
		}

		var payload DashboardRefreshPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			g.Log().Errorf(ctx, "Failed to unmarshal dashboard refresh payload: %v", err)
			return nil // don't requeue malformed messages
		}

		if payload.UserId == "" {
			g.Log().Warning(ctx, "Dashboard refresh message with empty userId, skipping")
			return nil
		}

		g.Log().Debugf(ctx, "Received dashboard refresh event: userId=%s reason=%s", payload.UserId, payload.Reason)

		// Debounce: if a timer already exists for this user, reset it.
		// This coalesces rapid mutations (e.g. bulk import of 500 transactions)
		// into a single snapshot rebuild.
		pendingMu.Lock()
		if timer, ok := pending[payload.UserId]; ok {
			timer.Stop()
		}
		pending[payload.UserId] = time.AfterFunc(debounceInterval, func() {
			pendingMu.Lock()
			delete(pending, payload.UserId)
			pendingMu.Unlock()

			if err := RebuildSnapshots(context.Background(), payload.UserId); err != nil {
				g.Log().Errorf(context.Background(), "Dashboard snapshot rebuild failed for user %s: %v", payload.UserId, err)
			}
		})
		pendingMu.Unlock()

		return nil
	})
}

// StartSnapshotFlushTicker runs a periodic ticker that flushes all dashboard
// snapshots from Redis to the database. The interval is controlled by the
// cache.snapshot_flush.ttl config (default 24 hours / T+1).
// This function blocks until ctx is cancelled; call it as a goroutine.
func StartSnapshotFlushTicker(ctx context.Context) {
	interval := utils.CacheTTL.SnapshotFlush
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	g.Log().Infof(ctx, "Starting snapshot flush ticker with interval: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.Log().Info(ctx, "Snapshot flush ticker stopped")
			return
		case <-ticker.C:
			g.Log().Info(ctx, "Snapshot flush ticker fired, persisting all snapshots to DB...")
			FlushAllSnapshotsToDB(ctx)
			g.Log().Info(ctx, "Snapshot flush completed")
		}
	}
}
