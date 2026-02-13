package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// processAccountMigration handles account migration task
func (s *sTask) processAccountMigration(ctx context.Context, payload json.RawMessage) error {
	var data struct {
		TaskId  string                        `json:"taskId"`
		Payload model.AccountMigrationPayload `json:"payload"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return gerror.Wrap(err, "failed to unmarshal payload")
	}

	taskId, err := uuid.Parse(data.TaskId)
	if err != nil {
		return gerror.Wrap(err, "invalid task ID")
	}
	migrationPayload := data.Payload

	// Update task status to running
	_, err = dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, taskId).Data(entity.Tasks{
		Status:    model.TaskStatusRunning,
		StartedAt: gtime.Now(),
	}).Update()
	if err != nil {
		return gerror.Wrap(err, "failed to update task status")
	}

	// Check if task was cancelled
	task, err := s.GetTask(ctx, taskId)
	if err != nil || task.Status == model.TaskStatusCancelled {
		return nil
	}

	// Execute migration in transaction
	result := model.AccountMigrationResult{}
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		return s.executeMigration(ctx, tx, taskId, &migrationPayload, &result)
	})

	if err != nil {
		s.FailTask(ctx, taskId, err.Error())
		return err
	}

	// Trigger dashboard snapshot rebuild after migration completes
	if migrationPayload.Payload != nil {
		dashboard.PublishDashboardRefresh(ctx, migrationPayload.Payload.UserId.String(), "account_migration")
	}

	return s.CompleteTask(ctx, taskId, result)
}

// executeMigration performs the actual migration within a transaction
// It uses transaction-based balance transfer to preserve the accounting equation:
// Assets = Liabilities + Equity
func (s *sTask) executeMigration(ctx context.Context, tx gdb.TX, taskId uuid.UUID, payload *model.AccountMigrationPayload, result *model.AccountMigrationResult) error {
	accountIds := append([]uuid.UUID{payload.AccountId}, payload.ChildAccountIds...)
	batchSize := 500

	// Count total transactions to migrate
	var totalCount int
	for _, accId := range accountIds {
		count, _ := tx.Model(dao.Transactions.Table()).
			Where(dao.Transactions.Columns().FromAccountId+" = ? OR "+dao.Transactions.Columns().ToAccountId+" = ?", accId, accId).
			Count()
		totalCount += count
	}

	// Update total items
	tx.Model(dao.Tasks.Table()).Where(dao.Tasks.Columns().Id, taskId).Data(g.Map{dao.Tasks.Columns().TotalItems: totalCount}).Update()

	processed := 0

	// Migrate transactions for each account
	for _, accId := range accountIds {
		// Get source account details
		var acc entity.Accounts
		if err := tx.Model(dao.Accounts.Table()).Where(dao.Accounts.Columns().Id, accId).Scan(&acc); err != nil {
			return gerror.Wrapf(err, "failed to get account %s", accId)
		}

		targetId := payload.MigrationTargets[acc.CurrencyCode]
		if targetId == uuid.Nil {
			g.Log().Warningf(ctx, "No migration target for currency %s, skipping account %s", acc.CurrencyCode, accId)
			continue
		}

		// Get target account details
		var targetAcc entity.Accounts
		if err := tx.Model(dao.Accounts.Table()).Where(dao.Accounts.Columns().Id, targetId).Scan(&targetAcc); err != nil {
			return gerror.Wrapf(err, "failed to get target account %s", targetId)
		}

		// Step 1: Update transaction references (from_account_id)
		fromResult, err := tx.Model(dao.Transactions.Table()).
			Where(dao.Transactions.Columns().FromAccountId, accId).
			Data(g.Map{dao.Transactions.Columns().FromAccountId: targetId}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to update from_account_id")
		}
		fromCount, _ := fromResult.RowsAffected()
		result.TransactionsMigrated += int(fromCount)

		// Step 2: Update transaction references (to_account_id)
		toResult, err := tx.Model(dao.Transactions.Table()).
			Where(dao.Transactions.Columns().ToAccountId, accId).
			Data(g.Map{dao.Transactions.Columns().ToAccountId: targetId}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to update to_account_id")
		}
		toCount, _ := toResult.RowsAffected()
		result.TransactionsMigrated += int(toCount)

		// Step 3: Transfer balance using transactions (for Asset/Liability accounts only)
		// This preserves the accounting equation: Assets = Liabilities + Equity
		if acc.Type == utils.AccountTypeAsset || acc.Type == utils.AccountTypeLiability {
			if err := s.migrateBalanceWithTransactions(ctx, tx, &acc, &targetAcc, payload.UserId); err != nil {
				return gerror.Wrapf(err, "failed to migrate balance from %s to %s", acc.Name, targetAcc.Name)
			}
		}
		result.BalancesMerged++

		// Step 4: Soft delete the source account
		_, err = tx.Model(dao.Accounts.Table()).
			Where(dao.Accounts.Columns().Id, accId).
			Data(g.Map{dao.Accounts.Columns().DeletedAt: gtime.Now()}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to soft delete account")
		}
		result.AccountsDeleted++

		// Step 5: Invalidate cache for affected accounts
		_ = utils.InvalidateCache(ctx, utils.AccountCacheKey(accId.String()))
		_ = utils.InvalidateCache(ctx, utils.AccountCacheKey(targetId.String()))

		// Update progress
		processed += int(fromCount) + int(toCount)
		progress := 0
		if totalCount > 0 {
			progress = (processed * 100) / totalCount
		}
		tx.Model(dao.Tasks.Table()).Where(dao.Tasks.Columns().Id, taskId).Data(g.Map{
			dao.Tasks.Columns().Progress:       progress,
			dao.Tasks.Columns().ProcessedItems: processed,
		}).Update()

		// Check for cancellation periodically
		var taskStatus int
		tx.Model(dao.Tasks.Table()).Where(dao.Tasks.Columns().Id, taskId).Fields(dao.Tasks.Columns().Status).Scan(&taskStatus)
		if taskStatus == model.TaskStatusCancelled {
			return gerror.New("task cancelled by user")
		}

		// Log progress for large batches
		if processed%batchSize == 0 {
			g.Log().Debugf(ctx, "Migration progress: %d/%d", processed, totalCount)
		}
	}

	return nil
}

// migrateBalanceWithTransactions transfers the balance from source account to target account
// using proper double-entry bookkeeping via equity account.
// This ensures the accounting equation (Assets = Liabilities + Equity) is always maintained.
//
// Flow:
// 1. Source Account → Equity Account (clear source balance)
// 2. Equity Account → Target Account (add to target balance)
func (s *sTask) migrateBalanceWithTransactions(ctx context.Context, tx gdb.TX, source *entity.Accounts, target *entity.Accounts, userId uuid.UUID) error {
	sourceBalance := utils.NewFromEntity(source)

	// Skip if source balance is zero
	if sourceBalance.IsZero() {
		g.Log().Debugf(ctx, "Source account %s has zero balance, skipping balance migration", source.Name)
		return nil
	}

	// Get or create equity account for this currency
	equityAccountId, err := s.getOrCreateMigrationEquityAccountInTx(ctx, tx, source.CurrencyCode, userId)
	if err != nil {
		return gerror.Wrap(err, "failed to get/create equity account")
	}

	units, nanos := sourceBalance.ToEntityValues()
	now := gtime.Now()
	noteSourceToEquity := fmt.Sprintf("Balance Migration - %s → Equity", source.Name)
	noteEquityToTarget := fmt.Sprintf("Balance Migration - Equity → %s", target.Name)

	// Transaction 1: Source Account → Equity Account (clear source balance)
	tx1Id, err := uuid.NewV7()
	if err != nil {
		return gerror.Wrap(err, "failed to generate UUID for transaction 1")
	}

	tx1 := entity.Transactions{
		Id:            tx1Id,
		UserId:        userId,
		FromAccountId: source.Id,
		ToAccountId:   equityAccountId,
		BalanceUnits:  units,
		BalanceNanos:  int(nanos),
		CurrencyCode:  source.CurrencyCode,
		Date:          now,
		Note:          noteSourceToEquity,
		Type:          utils.TransactionTypeTransfer,
	}

	_, err = tx.Model(dao.Transactions.Table()).
		FieldsEx(dao.Transactions.Columns().BalanceDecimal, dao.Transactions.Columns().DeletedAt).
		Data(tx1).
		Insert()
	if err != nil {
		return gerror.Wrap(err, "failed to insert source-to-equity transaction")
	}

	// Apply balance change: decrease source, increase equity
	if err := service.Balance().ApplyTransactionInTx(ctx, tx, &model.TransactionCreateInput{
		UserId:        userId,
		FromAccountId: source.Id,
		ToAccountId:   equityAccountId,
		BalanceUnits:  units,
		BalanceNanos:  int(nanos),
		CurrencyCode:  source.CurrencyCode,
		Type:          utils.TransactionTypeTransfer,
	}); err != nil {
		return gerror.Wrap(err, "failed to apply source-to-equity balance change")
	}

	// Transaction 2: Equity Account → Target Account (add to target balance)
	tx2Id, err := uuid.NewV7()
	if err != nil {
		return gerror.Wrap(err, "failed to generate UUID for transaction 2")
	}

	tx2 := entity.Transactions{
		Id:            tx2Id,
		UserId:        userId,
		FromAccountId: equityAccountId,
		ToAccountId:   target.Id,
		BalanceUnits:  units,
		BalanceNanos:  int(nanos),
		CurrencyCode:  source.CurrencyCode,
		Date:          now,
		Note:          noteEquityToTarget,
		Type:          utils.TransactionTypeTransfer,
	}

	_, err = tx.Model(dao.Transactions.Table()).
		FieldsEx(dao.Transactions.Columns().BalanceDecimal, dao.Transactions.Columns().DeletedAt).
		Data(tx2).
		Insert()
	if err != nil {
		return gerror.Wrap(err, "failed to insert equity-to-target transaction")
	}

	// Apply balance change: decrease equity, increase target
	if err := service.Balance().ApplyTransactionInTx(ctx, tx, &model.TransactionCreateInput{
		UserId:        userId,
		FromAccountId: equityAccountId,
		ToAccountId:   target.Id,
		BalanceUnits:  units,
		BalanceNanos:  int(nanos),
		CurrencyCode:  source.CurrencyCode,
		Type:          utils.TransactionTypeTransfer,
	}); err != nil {
		return gerror.Wrap(err, "failed to apply equity-to-target balance change")
	}

	g.Log().Infof(ctx, "Successfully migrated balance from %s to %s via equity account (units=%d, nanos=%d)",
		source.Name, target.Name, units, nanos)

	return nil
}

// getOrCreateMigrationEquityAccountInTx gets or creates a migration equity account within a transaction.
// This is a dedicated equity account for balance migrations to maintain audit trail.
func (s *sTask) getOrCreateMigrationEquityAccountInTx(ctx context.Context, tx gdb.TX, currency string, userId uuid.UUID) (uuid.UUID, error) {
	equityAccountName := "Migration Equity - " + currency

	var existing entity.Accounts
	err := tx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().UserId, userId).
		Where(dao.Accounts.Columns().Type, utils.AccountTypeEquity).
		Where(dao.Accounts.Columns().CurrencyCode, currency).
		Where(dao.Accounts.Columns().Name, equityAccountName).
		Where(dao.Accounts.Columns().DeletedAt + " IS NULL").
		Scan(&existing)

	if err != nil && err != sql.ErrNoRows {
		g.Log().Errorf(ctx, "Failed to scan equity account: %v", err)
		return uuid.Nil, fmt.Errorf("failed to query existing equity account: %w", err)
	}

	// If found, return its ID
	if existing.Id != uuid.Nil {
		return existing.Id, nil
	}

	// Create new equity account for migrations
	newId, err := uuid.NewV7()
	if err != nil {
		g.Log().Errorf(ctx, "Failed to generate UUID: %v", err)
		return uuid.Nil, gerror.Wrap(err, "failed to generate UUID")
	}

	equityAccount := entity.Accounts{
		Id:           newId,
		UserId:       userId,
		Name:         equityAccountName,
		Type:         utils.AccountTypeEquity,
		IsGroup:      false,
		BalanceUnits: 0,
		BalanceNanos: 0,
		CurrencyCode: currency,
		Date:         gtime.Now(),
	}

	_, err = tx.Model(dao.Accounts.Table()).
		FieldsEx(
			dao.Accounts.Columns().BalanceDecimal,
			dao.Accounts.Columns().ParentId,
			dao.Accounts.Columns().DefaultChildId,
		).
		Insert(equityAccount)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to insert equity account: %v", err)
		return uuid.Nil, gerror.Wrap(err, "failed to create equity account")
	}

	g.Log().Infof(ctx, "Created migration equity account: %s (id=%s)", equityAccountName, newId)
	return newId, nil
}
