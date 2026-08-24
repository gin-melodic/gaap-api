package boot

import (
	"context"
	"fmt"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/redis"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/shopspring/decimal"
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

	// After balance sync, warm dashboard snapshots for all users
	go WarmDashboardSnapshots(ctx)
}

// acquireDistributedLock tries to acquire a Redis distributed lock.
// Returns true if lock acquired, false otherwise.
func acquireDistributedLock(ctx context.Context) bool {
	redisClient, err := redis.GetRedisClient(ctx, redis.RedisTypeSyncLock)
	if err != nil {
		g.Log().Warning(ctx, "Redis not configured. Running balance sync without distributed lock.")
		return true // Proceed without lock if Redis not available
	}

	// SET key value NX PX milliseconds
	// NX: Only set if not exists
	// PX: Expire in milliseconds
	result, err := redisClient.Do(ctx, "SET", balanceSyncLockKey, balanceSyncLockValue, "NX", "PX", balanceSyncLockTimeout)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to acquire Redis lock: %v. Proceeding without lock.", err)
		return true // Proceed without lock on error
	}

	// SET NX returns "OK" if successful, nil if key already exists
	return result.String() == "OK"
}

// releaseDistributedLock releases the Redis distributed lock.
func releaseDistributedLock(ctx context.Context) {
	redisClient, err := redis.GetRedisClient(ctx, redis.RedisTypeSyncLock)
	if err != nil {
		g.Log().Warning(ctx, "Redis not configured. Running balance sync without distributed lock.")
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
	_, err = redisClient.Do(ctx, "EVAL", luaScript, 1, balanceSyncLockKey, balanceSyncLockValue)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to release Redis lock: %v", err)
	}
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
		expectedMoney, err := calculateExpectedBalance(account.Id.String())
		if err != nil {
			g.Log().Warningf(ctx, "Failed to calculate an account balance: %v", err)
			continue
		}

		accountMoney := utils.NewFromEntity(&account)

		// Check if balance needs update
		if !accountMoney.Equals(expectedMoney) {
			// Update balance
			expectedBalanceUnits, expectedBalanceNanos := expectedMoney.ToEntityValues()
			_, err = g.DB().Model(dao.Accounts.Table()).
				Where(dao.Accounts.Columns().Id, account.Id).
				Data(g.Map{
					dao.Accounts.Columns().BalanceUnits: expectedBalanceUnits,
					dao.Accounts.Columns().BalanceNanos: expectedBalanceNanos,
				}).
				Update()
			if err != nil {
				g.Log().Warningf(ctx, "Failed to update an account balance: %v", err)
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
func calculateExpectedBalance(accountId string) (*utils.MoneyHelper, error) {
	balance := &utils.MoneyHelper{
		Decimal:  decimal.NewFromInt(0),
		Currency: "",
	}

	// Sum of INCOME transactions where this account is the to_account
	// INCOME: money comes into to_account
	var incomeTransactions []entity.Transactions
	err := g.DB().Model(dao.Transactions.Table()).
		Where("to_account_id", accountId).
		Where("type", TypeIncome).
		Where("deleted_at IS NULL").
		Scan(&incomeTransactions)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get income transactions")
	}
	incomeSum := &utils.MoneyHelper{
		Decimal:  decimal.NewFromInt(0),
		Currency: "",
	}
	for _, t := range incomeTransactions {
		incomeSum, err = incomeSum.Add(utils.NewFromTransactions(&t))
	}
	if err != nil {
		return nil, gerror.Wrap(err, "failed to sum income")
	}

	balance, err = balance.Add(incomeSum)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to add income")
	}

	// Sum of TRANSFER transactions where this account is the to_account
	// TRANSFER: money comes into to_account
	var transferInTransactions []entity.Transactions
	err = g.DB().Model(dao.Transactions.Table()).
		Where("to_account_id", accountId).
		Where("type", TypeTransfer).
		Where("deleted_at IS NULL").
		Scan(&transferInTransactions)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get transfer in transactions")
	}
	transferInSum := &utils.MoneyHelper{
		Decimal:  decimal.NewFromInt(0),
		Currency: "",
	}
	for _, t := range transferInTransactions {
		transferInSum, err = transferInSum.Add(utils.NewFromTransactions(&t))
	}
	if err != nil {
		return nil, gerror.Wrap(err, "failed to sum transfer in")
	}

	balance, err = balance.Add(transferInSum)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to add transfer in")
	}

	// Sum of EXPENSE transactions where this account is the from_account
	// EXPENSE: money goes out from from_account
	var expenseTransactions []entity.Transactions
	err = g.DB().Model(dao.Transactions.Table()).
		Where("from_account_id", accountId).
		Where("type", TypeExpense).
		Where("deleted_at IS NULL").
		Scan(&expenseTransactions)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get expense transactions")
	}
	expenseSum := &utils.MoneyHelper{
		Decimal:  decimal.NewFromInt(0),
		Currency: "",
	}
	for _, t := range expenseTransactions {
		expenseSum, err = expenseSum.Add(utils.NewFromTransactions(&t))
	}
	if err != nil {
		return nil, gerror.Wrap(err, "failed to sum expense")
	}

	balance, err = balance.Sub(expenseSum)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to sub expense")
	}

	return balance, nil
}

// WarmDashboardSnapshots rebuilds dashboard snapshots from transactional data.
// Persisted snapshots are a recovery cache, not a source of truth: restoring them
// blindly can resurrect data that predates the latest committed transaction.
func WarmDashboardSnapshots(ctx context.Context) {
	g.Log().Info(ctx, "Warming dashboard snapshots for all users...")

	var users []struct {
		Id string `orm:"id"`
	}
	err := dao.Users.Ctx(ctx).
		Fields(dao.Users.Columns().Id).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&users)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to query users for dashboard warmup: %v", err)
		return
	}

	for _, u := range users {
		if err := dashboard.RebuildSnapshots(ctx, u.Id); err != nil {
			g.Log().Warningf(ctx, "Failed to warm dashboard snapshot for user %s: %v", u.Id, err)
		}
	}

	g.Log().Infof(ctx, "Dashboard warmup completed: %d users rebuilt from source", len(users))
}
