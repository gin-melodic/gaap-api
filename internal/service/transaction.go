// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/google/uuid"
)

type (
	ITransaction interface {
		ListTransactions(ctx context.Context, in model.TransactionQueryInput) (out []entity.Transactions, total int, err error)
		// CreateTransaction creates a new transaction.
		// If tx is provided, it will be used for the transaction.
		CreateTransaction(ctx context.Context, in model.TransactionCreateInput, tx gdb.TX) (out *entity.Transactions, err error)
		// GetTransaction returns a transaction by ID with caching.
		GetTransaction(ctx context.Context, id uuid.UUID) (out *entity.Transactions, err error)
		UpdateTransaction(ctx context.Context, id uuid.UUID, in model.TransactionUpdateInput) (out *entity.Transactions, err error)
		DeleteTransaction(ctx context.Context, id uuid.UUID) (err error)
	}
)

var (
	localTransaction ITransaction
)

func Transaction() ITransaction {
	if localTransaction == nil {
		panic("implement not found for interface ITransaction, forgot register?")
	}
	return localTransaction
}

func RegisterTransaction(i ITransaction) {
	localTransaction = i
}
