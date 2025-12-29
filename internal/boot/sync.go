package boot

import (
	"context"
	"fmt"
	"os"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	// Redis distributed lock key for balance sync
	balanceSyncLockKey = "gaap:balance:sync:lock"
	// Lock timeout in milliseconds (5 minutes)
	balanceSyncLockTimeout = 5 * 60 * 1000
	// Unique lock value for this instance
	balanceSyncLockValue = "balance-sync-lock"
)

// Transaction types (same as in balance package)
const (
	TypeExpense  = "EXPENSE"
	TypeIncome   = "INCOME"
	TypeTransfer = "TRANSFER"
)

// SyncBalances synchronizes all account balances based on transactions.
// Uses Redis distributed lock to ensure only one instance runs this in a distributed environment.
func SyncBalances(ctx context.Context) {
	g.Log().Info(ctx, "Starting balance synchronization check...")

	// Try to acquire distributed lock
	if !acquireDistributedLock(ctx) {
		g.Log().Info(ctx, "Another instance is running balance sync. Skipping...")
		return
	}
	defer releaseDistributedLock(ctx)

	g.Log().Info(ctx, "Acquired distributed lock. Running balance sync...")

	// Perform balance synchronization
	if err := performBalanceSync(ctx); err != nil {
		g.Log().Errorf(ctx, "Failed to sync balances: %v", err)
		return
	}

	g.Log().Info(ctx, "Balance synchronization completed successfully.")
}

// acquireDistributedLock tries to acquire a Redis distributed lock.
// Returns true if lock acquired, false otherwise.
func acquireDistributedLock(ctx context.Context) bool {
	redis := getRedisClient(ctx)
	if redis == nil {
		g.Log().Warning(ctx, "Redis not configured. Running balance sync without distributed lock.")
		return true // Proceed without lock if Redis not available
	}

	// SET key value NX PX milliseconds
	// NX: Only set if not exists
	// PX: Expire in milliseconds
	result, err := redis.Do(ctx, "SET", balanceSyncLockKey, balanceSyncLockValue, "NX", "PX", balanceSyncLockTimeout)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to acquire Redis lock: %v. Proceeding without lock.", err)
		return true // Proceed without lock on error
	}

	// SET NX returns "OK" if successful, nil if key already exists
	return result.String() == "OK"
}

// releaseDistributedLock releases the Redis distributed lock.
func releaseDistributedLock(ctx context.Context) {
	redis := getRedisClient(ctx)
	if redis == nil {
		return
	}

	// Only delete if we own the lock (check value matches)
	// Use Lua script for atomic check-and-delete
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := redis.Do(ctx, "EVAL", luaScript, 1, balanceSyncLockKey, balanceSyncLockValue)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to release Redis lock: %v", err)
	}
}

// getRedisClient returns the Redis client if configured.
func getRedisClient(ctx context.Context) *gredis.Redis {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		return nil
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	password := os.Getenv("REDIS_PASSWORD")

	config := &gredis.Config{
		Address: fmt.Sprintf("%s:%s", host, port),
		Pass:    password,
		Db:      0, // Use db 0 for locks
	}

	redis, err := gredis.New(config)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to create Redis client: %v", err)
		return nil
	}

	// Test connection
	if _, err := redis.Do(ctx, "PING"); err != nil {
		g.Log().Warningf(ctx, "Failed to connect to Redis: %v", err)
		return nil
	}

	return redis
}

// performBalanceSync recalculates and updates all account balances.
func performBalanceSync(ctx context.Context) error {
	startTime := time.Now()

	// Get all accounts
	var accounts []entity.Accounts
	err := g.DB().Model(dao.Accounts.Table()).
		Where("deleted_at IS NULL").
		Scan(&accounts)
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	if len(accounts) == 0 {
		g.Log().Info(ctx, "No accounts found. Skipping balance sync.")
		return nil
	}

	g.Log().Infof(ctx, "Found %d accounts. Calculating expected balances...", len(accounts))

	// Calculate expected balance for each account
	updatedCount := 0
	for _, account := range accounts {
		expectedBalance, err := calculateExpectedBalance(ctx, account.Id)
		if err != nil {
			g.Log().Warningf(ctx, "Failed to calculate balance for account %s: %v", account.Id, err)
			continue
		}

		// Check if balance needs update
		if account.Balance != expectedBalance {
			g.Log().Infof(ctx, "Account %s (%s): current=%.2f, expected=%.2f. Updating...",
				account.Id, account.Name, account.Balance, expectedBalance)

			// Update balance
			_, err = g.DB().Model(dao.Accounts.Table()).
				Where("id", account.Id).
				Data(g.Map{"balance": expectedBalance}).
				Update()
			if err != nil {
				g.Log().Warningf(ctx, "Failed to update balance for account %s: %v", account.Id, err)
				continue
			}
			updatedCount++
		}
	}

	duration := time.Since(startTime)
	g.Log().Infof(ctx, "Balance sync completed in %v. Updated %d/%d accounts.",
		duration, updatedCount, len(accounts))

	return nil
}

// calculateExpectedBalance calculates the expected balance for an account based on transactions.
func calculateExpectedBalance(ctx context.Context, accountId string) (float64, error) {
	var balance float64 = 0

	// Sum of INCOME transactions where this account is the to_account
	// INCOME: money comes into to_account
	var incomeSum struct {
		Total float64
	}
	err := g.DB().Model(dao.Transactions.Table()).
		Fields("COALESCE(SUM(amount), 0) as total").
		Where("to_account_id", accountId).
		Where("type", TypeIncome).
		Where("deleted_at IS NULL").
		Scan(&incomeSum)
	if err != nil {
		return 0, fmt.Errorf("failed to sum income: %w", err)
	}
	balance += incomeSum.Total

	// Sum of TRANSFER transactions where this account is the to_account
	// TRANSFER: money comes into to_account
	var transferInSum struct {
		Total float64
	}
	err = g.DB().Model(dao.Transactions.Table()).
		Fields("COALESCE(SUM(amount), 0) as total").
		Where("to_account_id", accountId).
		Where("type", TypeTransfer).
		Where("deleted_at IS NULL").
		Scan(&transferInSum)
	if err != nil {
		return 0, fmt.Errorf("failed to sum transfer in: %w", err)
	}
	balance += transferInSum.Total

	// Sum of EXPENSE transactions where this account is the from_account
	// EXPENSE: money goes out from from_account
	var expenseSum struct {
		Total float64
	}
	err = g.DB().Model(dao.Transactions.Table()).
		Fields("COALESCE(SUM(amount), 0) as total").
		Where("from_account_id", accountId).
		Where("type", TypeExpense).
		Where("deleted_at IS NULL").
		Scan(&expenseSum)
	if err != nil {
		return 0, fmt.Errorf("failed to sum expense: %w", err)
	}
	balance -= expenseSum.Total

	// Sum of TRANSFER transactions where this account is the from_account
	// TRANSFER: money goes out from from_account
	var transferOutSum struct {
		Total float64
	}
	err = g.DB().Model(dao.Transactions.Table()).
		Fields("COALESCE(SUM(amount), 0) as total").
		Where("from_account_id", accountId).
		Where("type", TypeTransfer).
		Where("deleted_at IS NULL").
		Scan(&transferOutSum)
	if err != nil {
		return 0, fmt.Errorf("failed to sum transfer out: %w", err)
	}
	balance -= transferOutSum.Total

	return balance, nil
}
