package service

import (
	"context"
	"gaap-api/internal/model"
)

type ITransaction interface {
	ListTransactions(ctx context.Context, in model.TransactionQueryInput) (out []model.Transaction, total int, err error)
	CreateTransaction(ctx context.Context, in model.TransactionCreateInput) (out *model.Transaction, err error)
	GetTransaction(ctx context.Context, id string) (out *model.Transaction, err error)
	UpdateTransaction(ctx context.Context, id string, in model.TransactionUpdateInput) (out *model.Transaction, err error)
	DeleteTransaction(ctx context.Context, id string) (err error)
}

var localTransaction ITransaction

func Transaction() ITransaction {
	if localTransaction == nil {
		panic("implement not found for interface ITransaction, forgot register?")
	}
	return localTransaction
}

func RegisterTransaction(i ITransaction) {
	localTransaction = i
}
