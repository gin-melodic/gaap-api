package balance

import (
	"context"
	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

type sBalance struct{}

func init() {
	service.RegisterBalance(New())
}

func New() *sBalance {
	return &sBalance{}
}

// ApplyTransaction applies the balance changes for a transaction.
func (s *sBalance) ApplyTransaction(ctx context.Context, tx *model.TransactionCreateInput) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		return s.ApplyTransactionInTx(ctx, dbTx, tx)
	})
}

// ApplyTransactionInTx applies balance changes within an existing transaction.
func (s *sBalance) ApplyTransactionInTx(ctx context.Context, dbTx gdb.TX, tx *model.TransactionCreateInput) error {
	switch tx.Type {
	case utils.TransactionTypeExpense:
		// EXPENSE: money goes out from asset account, into expense account
		// Decrease from_account balance
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to decrease from_account balance")
		}
		// Increase to_account balance (Expense Category Account)
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to increase to_account balance")
		}
	case utils.TransactionTypeIncome:
		// INCOME: money comes in to asset account
		// Increase to_account balance
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to increase to_account balance")
		}
		// Decrease from_account balance (Income source account)
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to decrease from_account balance")
		}
	case utils.TransactionTypeTransfer:
		// TRANSFER: money moves between accounts
		// Decrease from_account, increase to_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to decrease from_account balance")
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to increase to_account balance")
		}
	case utils.TransactionTypeOpeningBalance:
		// OPENING_BALANCE: from equity account to target account
		// Decrease from_account (equity) and increase to_account (asset/liability)
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to decrease equity account balance")
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to increase target account balance")
		}
	default:
		return gerror.Newf("unknown transaction type: %d", tx.Type)
	}
	return nil
}

// ReverseTransaction reverses the balance changes for a transaction.
func (s *sBalance) ReverseTransaction(ctx context.Context, tx *model.Transaction) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		return s.ReverseTransactionInTx(ctx, dbTx, tx)
	})
}

// ReverseTransactionInTx reverses balance changes within an existing transaction.
func (s *sBalance) ReverseTransactionInTx(ctx context.Context, dbTx gdb.TX, tx *model.Transaction) error {
	switch tx.Type {
	case utils.TransactionTypeExpense:
		// Reverse EXPENSE: restore money to from_account, decrease expense account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to restore from_account balance")
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to decrease to_account balance")
		}
	case utils.TransactionTypeIncome:
		// Reverse INCOME: remove money from to_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to reverse to_account balance")
		}
		// Restore from_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to restore from_account balance")
		}
	case utils.TransactionTypeTransfer:
		// Reverse TRANSFER: restore from_account, decrease to_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to restore from_account balance")
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to reverse to_account balance")
		}
	case utils.TransactionTypeOpeningBalance:
		// Reverse OPENING_BALANCE: restore equity account, decrease target account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.FromAccountId, tx.BalanceUnits, tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to restore equity account balance")
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.ToAccountId, -tx.BalanceUnits, -tx.BalanceNanos, tx.CurrencyCode); err != nil {
			return gerror.Wrap(err, "failed to reverse target account balance")
		}
	default:
		return gerror.Newf("unknown transaction type: %d", tx.Type)
	}
	return nil
}

// UpdateAccountBalance directly updates an account's balance by a delta.
func (s *sBalance) UpdateAccountBalance(ctx context.Context, accountId uuid.UUID, deltaUnits int64, deltaNanos int, currency string) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		return s.UpdateAccountBalanceInTx(ctx, dbTx, accountId, deltaUnits, deltaNanos, currency)
	})
}

// UpdateAccountBalanceInTx updates balance within an existing transaction.
func (s *sBalance) UpdateAccountBalanceInTx(ctx context.Context, dbTx gdb.TX, accountId uuid.UUID, deltaUnits int64, deltaNanos int, currency string) error {
	return s.updateBalanceInTx(ctx, dbTx, accountId, deltaUnits, deltaNanos, currency)
}

// updateBalanceInTx is the internal method that performs the actual balance update.
// It uses SELECT FOR UPDATE to prevent concurrent modification issues.
// Uses MoneyHelper for safe decimal arithmetic on units/nanos.
func (s *sBalance) updateBalanceInTx(ctx context.Context, dbTx gdb.TX, accountId uuid.UUID, deltaUnits int64, deltaNanos int, currency string) error {
	if accountId == uuid.Nil {
		return nil // Skip if no account specified
	}

	if deltaUnits == 0 && deltaNanos == 0 {
		return nil // Skip if no change
	}

	// Get current balance with row lock (SELECT FOR UPDATE)
	var account entity.Accounts
	err := dbTx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().Id, accountId).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		LockUpdate().
		Scan(&account)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to load account balance: %v", err)
		return gerror.Wrapf(err, "failed to get account %s", accountId)
	}

	if account.Id == uuid.Nil {
		return gerror.New("account not found during balance update")
	}

	// Use MoneyHelper for safe arithmetic
	currentBalance := utils.NewFromEntity(&account)

	// Create delta MoneyHelper manually from units/nanos
	var deltaBalance *utils.MoneyHelper
	// Convert delta units/nanos to MoneyHelper
	deltaEntity := &entity.Accounts{
		BalanceUnits: deltaUnits,
		BalanceNanos: deltaNanos,
		CurrencyCode: currency,
	}
	deltaBalance = utils.NewFromEntity(deltaEntity)

	// Perform addition
	newBalance, err := currentBalance.Add(deltaBalance)
	if err != nil {
		return gerror.Wrap(err, "currency mismatch during balance update")
	}

	newUnits, newNanos := newBalance.ToEntityValues()

	// Update balance using entity struct
	result, err := dbTx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().Id, accountId).
		Data(g.Map{
			dao.Accounts.Columns().BalanceUnits: newUnits,
			dao.Accounts.Columns().BalanceNanos: int(newNanos),
		}).
		Update()
	if err != nil {
		g.Log().Errorf(ctx, "Failed to update account balance: %v", err)
		return gerror.Wrapf(err, "failed to update balance for account %s", accountId)
	}
	rows, rowsErr := result.RowsAffected()
	if rowsErr != nil || rows != 1 {
		return gerror.New("balance update did not affect exactly one account")
	}

	return nil
}
