package service

import (
	"context"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
)

// IBalance defines the interface for balance management operations.
// This interface is designed to support future distributed implementations
// using RabbitMQ/Redis while currently using database transactions.
type IBalance interface {
	// ApplyTransaction applies the balance changes for a transaction.
	// For EXPENSE: decreases from_account balance
	// For INCOME: increases to_account balance
	// For TRANSFER: decreases from_account and increases to_account
	ApplyTransaction(ctx context.Context, tx *model.TransactionCreateInput) error

	// ApplyTransactionInTx applies balance changes within an existing transaction.
	ApplyTransactionInTx(ctx context.Context, dbTx gdb.TX, tx *model.TransactionCreateInput) error

	// ReverseTransaction reverses the balance changes for a transaction.
	// Used when updating or deleting transactions.
	ReverseTransaction(ctx context.Context, tx *model.Transaction) error

	// ReverseTransactionInTx reverses balance changes within an existing transaction.
	ReverseTransactionInTx(ctx context.Context, dbTx gdb.TX, tx *model.Transaction) error

	// UpdateAccountBalance directly updates an account's balance by a delta.
	// Positive delta increases balance, negative decreases.
	UpdateAccountBalance(ctx context.Context, accountId string, delta float64) error

	// UpdateAccountBalanceInTx updates balance within an existing transaction.
	UpdateAccountBalanceInTx(ctx context.Context, dbTx gdb.TX, accountId string, delta float64) error
}

var localBalance IBalance

func Balance() IBalance {
	if localBalance == nil {
		panic("implement not found for interface IBalance, forgot register?")
	}
	return localBalance
}

func RegisterBalance(i IBalance) {
	localBalance = i
}
