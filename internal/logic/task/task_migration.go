package task

import (
	"context"
	"encoding/json"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

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

	return s.CompleteTask(ctx, taskId, result)
}

// executeMigration performs the actual migration within a transaction
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
		// Get account currency
		var acc entity.Accounts
		tx.Model(dao.Accounts.Table()).Where(dao.Accounts.Columns().Id, accId).Scan(&acc)

		targetId := payload.MigrationTargets[acc.CurrencyCode]
		if targetId == uuid.Nil {
			continue
		}

		// Update from_account_id
		fromResult, err := tx.Model(dao.Transactions.Table()).
			Where(dao.Transactions.Columns().FromAccountId, accId).
			Data(g.Map{dao.Transactions.Columns().FromAccountId: targetId}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to update from_account_id")
		}
		fromCount, _ := fromResult.RowsAffected()
		result.TransactionsMigrated += int(fromCount)

		// Update to_account_id
		toResult, err := tx.Model(dao.Transactions.Table()).
			Where(dao.Transactions.Columns().ToAccountId, accId).
			Data(g.Map{dao.Transactions.Columns().ToAccountId: targetId}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to update to_account_id")
		}
		toCount, _ := toResult.RowsAffected()
		result.TransactionsMigrated += int(toCount)

		// Merge balance using MoneyHelper
		var targetAcc entity.Accounts
		tx.Model(dao.Accounts.Table()).Where(dao.Accounts.Columns().Id, targetId).Scan(&targetAcc)

		currentBalance := utils.NewFromEntity(&targetAcc)
		deltaBalance := utils.NewFromEntity(&acc)
		newBalance, _ := currentBalance.Add(deltaBalance)
		if newBalance != nil {
			newUnits, newNanos := newBalance.ToEntityValues()
			tx.Model(dao.Accounts.Table()).
				Where(dao.Accounts.Columns().Id, targetId).
				FieldsEx(dao.Accounts.Columns().BalanceDecimal).
				Data(entity.Accounts{
					BalanceUnits: newUnits,
					BalanceNanos: int(newNanos),
				}).
				Update()
		}
		result.BalancesMerged++

		// Soft delete account
		_, err = tx.Model(dao.Accounts.Table()).
			Where(dao.Accounts.Columns().Id, accId).
			Data(g.Map{dao.Accounts.Columns().DeletedAt: gtime.Now()}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to soft delete account")
		}
		result.AccountsDeleted++

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

		// Small batch delay to prevent overwhelming
		if processed%batchSize == 0 {
			g.Log().Debugf(ctx, "Migration progress: %d/%d", processed, totalCount)
		}
	}

	return nil
}
