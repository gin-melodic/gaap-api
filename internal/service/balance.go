// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/google/uuid"
)

type (
	IBalance interface {
		// ApplyTransaction applies the balance changes for a transaction.
		ApplyTransaction(ctx context.Context, tx *model.TransactionCreateInput) error
		// ApplyTransactionInTx applies balance changes within an existing transaction.
		ApplyTransactionInTx(ctx context.Context, dbTx gdb.TX, tx *model.TransactionCreateInput) error
		// ReverseTransaction reverses the balance changes for a transaction.
		ReverseTransaction(ctx context.Context, tx *model.Transaction) error
		// ReverseTransactionInTx reverses balance changes within an existing transaction.
		ReverseTransactionInTx(ctx context.Context, dbTx gdb.TX, tx *model.Transaction) error
		// UpdateAccountBalance directly updates an account's balance by a delta.
		UpdateAccountBalance(ctx context.Context, accountId uuid.UUID, deltaUnits int64, deltaNanos int, currency string) error
		// UpdateAccountBalanceInTx updates balance within an existing transaction.
		UpdateAccountBalanceInTx(ctx context.Context, dbTx gdb.TX, accountId uuid.UUID, deltaUnits int64, deltaNanos int, currency string) error
	}
)

var (
	localBalance IBalance
)

func Balance() IBalance {
	if localBalance == nil {
		panic("implement not found for interface IBalance, forgot register?")
	}
	return localBalance
}

func RegisterBalance(i IBalance) {
	localBalance = i
}
