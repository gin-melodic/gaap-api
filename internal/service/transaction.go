// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

	"github.com/google/uuid"
)

type (
	ITransaction interface {
		ListTransactions(ctx context.Context, in model.TransactionQueryInput) (out []entity.Transactions, total int, err error)
		CreateTransaction(ctx context.Context, in model.TransactionCreateInput) (out *entity.Transactions, err error)
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
