package utils

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// CacheKey prefixes for different data types
const (
	CacheKeyThemes        = "config:themes"
	CacheKeyAccountTypes  = "config:account_types"
	CacheKeyUserPrefix    = "user"
	CacheKeyAccountPrefix = "account"
	CacheKeyTaskPrefix    = "task"
)

// UserCacheKey generates the cache key for a user profile
func UserCacheKey(userId string) string {
	return CacheKeyUserPrefix + ":" + userId
}

// TaskCacheKey generates the cache key for a task
func TaskCacheKey(taskId string) string {
	return CacheKeyTaskPrefix + ":" + taskId
}

// AccountCacheKey generates the cache key for an account
func AccountCacheKey(accountId string) string {
	return CacheKeyAccountPrefix + ":" + accountId
}

// TransactionCacheKey generates the cache key for a transaction
func TransactionCacheKey(transactionId string) string {
	return "transaction:" + transactionId
}

// DashboardSummaryCacheKey generates the cache key for dashboard summary
func DashboardSummaryCacheKey(userId string) string {
	return "dashboard:summary:" + userId
}

// DashboardMonthlyCacheKey generates the cache key for monthly stats
func DashboardMonthlyCacheKey(userId string) string {
	return "dashboard:monthly:" + userId
}

// CacheTTL holds cache expiration durations for different data types.
// These defaults can be overridden via InitCacheTTL from config.
var CacheTTL = struct {
	Config        time.Duration // Configuration data (themes, account types, currencies)
	User          time.Duration // User profile data
	Account       time.Duration // Account data
	Transaction   time.Duration // Transaction data
	Dashboard     time.Duration // Dashboard aggregations
	Search        time.Duration // Search results
	Task          time.Duration // Task data
	SnapshotFlush time.Duration // Dashboard snapshot DB flush interval
}{
	Config:        time.Hour * 24,   // Config data: 24 hours (rarely changes)
	User:          time.Hour,        // User data: 1 hour
	Account:       time.Hour * 2,    // Account data: 2 hours
	Transaction:   time.Minute * 30, // Transaction data: 30 minutes
	Dashboard:     time.Minute * 5,  // Dashboard: 5 minutes (frequently updated)
	Search:        time.Minute * 5,  // Search results: 5 minutes
	Task:          time.Minute * 10, // Task data: 10 minutes
	SnapshotFlush: time.Hour * 24,   // Snapshot flush: T+1 daily (configurable)
}

// InitCacheTTL loads cache TTL configuration from config file (optional).
// Falls back to defaults if config values are not set.
func InitCacheTTL(ctx context.Context) {
	if ttl := g.Cfg().MustGet(ctx, "cache.config.ttl", 86400).Int64(); ttl > 0 {
		CacheTTL.Config = time.Duration(ttl) * time.Second
	}
	if ttl := g.Cfg().MustGet(ctx, "cache.user.ttl", 3600).Int64(); ttl > 0 {
		CacheTTL.User = time.Duration(ttl) * time.Second
	}
	if ttl := g.Cfg().MustGet(ctx, "cache.account.ttl", 7200).Int64(); ttl > 0 {
		CacheTTL.Account = time.Duration(ttl) * time.Second
	}
	if ttl := g.Cfg().MustGet(ctx, "cache.transaction.ttl", 1800).Int64(); ttl > 0 {
		CacheTTL.Transaction = time.Duration(ttl) * time.Second
	}
	if ttl := g.Cfg().MustGet(ctx, "cache.dashboard.ttl", 300).Int64(); ttl > 0 {
		CacheTTL.Dashboard = time.Duration(ttl) * time.Second
	}
	if ttl := g.Cfg().MustGet(ctx, "cache.search.ttl", 300).Int64(); ttl > 0 {
		CacheTTL.Search = time.Duration(ttl) * time.Second
	}
	if ttlStr := os.Getenv("SNAPSHOT_FLUSH_TTL"); ttlStr != "" {
		if ttl, err := strconv.ParseInt(ttlStr, 10, 64); err == nil && ttl > 0 {
			CacheTTL.SnapshotFlush = time.Duration(ttl) * time.Second
		}
	}

	g.Log().Debugf(ctx, "Cache TTL initialized: Config=%v, User=%v, Account=%v, Transaction=%v, Dashboard=%v, SnapshotFlush=%v",
		CacheTTL.Config, CacheTTL.User, CacheTTL.Account, CacheTTL.Transaction, CacheTTL.Dashboard, CacheTTL.SnapshotFlush)
}
