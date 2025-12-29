package balance

import (
	"context"
	"fmt"
	"gaap-api/internal/dao"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// Transaction types
const (
	TypeExpense  = "EXPENSE"
	TypeIncome   = "INCOME"
	TypeTransfer = "TRANSFER"
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
	case TypeExpense:
		// EXPENSE: money goes out from asset account, into expense account
		// Decrease from_account balance
		if err := s.updateBalanceInTx(ctx, dbTx, tx.From, -tx.Amount); err != nil {
			return fmt.Errorf("failed to decrease from_account balance: %w", err)
		}
		// Increase to_account balance (Expense Category Account)
		if err := s.updateBalanceInTx(ctx, dbTx, tx.To, tx.Amount); err != nil {
			return fmt.Errorf("failed to increase to_account balance: %w", err)
		}
	case TypeIncome:
		// INCOME: money comes in to asset account
		// Increase to_account balance
		if err := s.updateBalanceInTx(ctx, dbTx, tx.To, tx.Amount); err != nil {
			return fmt.Errorf("failed to increase to_account balance: %w", err)
		}
		// Decrease from_account balance? Usually income source isn't an account we track balance for,
		// or it's an Equity/Income account where Crediting it increases it.
		// For now, let's assume we do update it if it exists.
		// If From is empty, updateBalanceInTx skips it.
		if err := s.updateBalanceInTx(ctx, dbTx, tx.From, -tx.Amount); err != nil {
			return fmt.Errorf("failed to decrease from_account balance: %w", err)
		}
	case TypeTransfer:
		// TRANSFER: money moves between accounts
		// Decrease from_account, increase to_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.From, -tx.Amount); err != nil {
			return fmt.Errorf("failed to decrease from_account balance: %w", err)
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.To, tx.Amount); err != nil {
			return fmt.Errorf("failed to increase to_account balance: %w", err)
		}
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.Type)
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
	case TypeExpense:
		// Reverse EXPENSE: restore money to from_account, decrease expense account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.From, tx.Amount); err != nil {
			return fmt.Errorf("failed to restore from_account balance: %w", err)
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.To, -tx.Amount); err != nil {
			return fmt.Errorf("failed to decrease to_account balance: %w", err)
		}
	case TypeIncome:
		// Reverse INCOME: remove money from to_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.To, -tx.Amount); err != nil {
			return fmt.Errorf("failed to reverse to_account balance: %w", err)
		}
		// Restore from_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.From, tx.Amount); err != nil {
			return fmt.Errorf("failed to restore from_account balance: %w", err)
		}

	case TypeTransfer:
		// Reverse TRANSFER: restore from_account, decrease to_account
		if err := s.updateBalanceInTx(ctx, dbTx, tx.From, tx.Amount); err != nil {
			return fmt.Errorf("failed to restore from_account balance: %w", err)
		}
		if err := s.updateBalanceInTx(ctx, dbTx, tx.To, -tx.Amount); err != nil {
			return fmt.Errorf("failed to reverse to_account balance: %w", err)
		}
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.Type)
	}
	return nil
}

// UpdateAccountBalance directly updates an account's balance by a delta.
func (s *sBalance) UpdateAccountBalance(ctx context.Context, accountId string, delta float64) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		return s.UpdateAccountBalanceInTx(ctx, dbTx, accountId, delta)
	})
}

// UpdateAccountBalanceInTx updates balance within an existing transaction.
func (s *sBalance) UpdateAccountBalanceInTx(ctx context.Context, dbTx gdb.TX, accountId string, delta float64) error {
	return s.updateBalanceInTx(ctx, dbTx, accountId, delta)
}

// updateBalanceInTx is the internal method that performs the actual balance update.
// It uses SELECT FOR UPDATE to prevent concurrent modification issues.
func (s *sBalance) updateBalanceInTx(ctx context.Context, dbTx gdb.TX, accountId string, delta float64) error {
	if accountId == "" {
		return nil // Skip if no account specified
	}

	if delta == 0 {
		return nil // Skip if no change
	}

	g.Log().Debugf(ctx, "Updating balance for account %s by %.2f", accountId, delta)

	// Get current balance with row lock (SELECT FOR UPDATE)
	var account entity.Accounts
	err := dbTx.Model(dao.Accounts.Table()).
		Where("id", accountId).
		Where("deleted_at IS NULL").
		LockUpdate().
		Scan(&account)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to get account %s: %v", accountId, err)
		return fmt.Errorf("failed to get account %s: %w", accountId, err)
	}

	if account.Id == "" {
		// Account not found - this is OK for category accounts (EXPENSE/INCOME types)
		// which don't need balance tracking
		g.Log().Warningf(ctx, "Account %s not found, skipping balance update", accountId)
		return nil
	}

	// Calculate new balance
	newBalance := account.Balance + delta

	g.Log().Debugf(ctx, "Account %s: current=%.2f, delta=%.2f, new=%.2f",
		accountId, account.Balance, delta, newBalance)

	// Update balance
	_, err = dbTx.Model(dao.Accounts.Table()).
		Where("id", accountId).
		Data(g.Map{
			"balance": newBalance,
		}).
		Update()
	if err != nil {
		g.Log().Errorf(ctx, "Failed to update balance for account %s: %v", accountId, err)
		return fmt.Errorf("failed to update balance for account %s: %w", accountId, err)
	}

	g.Log().Debugf(ctx, "Successfully updated balance for account %s to %.2f", accountId, newBalance)
	return nil
}
