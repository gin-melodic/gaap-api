package transaction

import (
	"context"
	v1 "gaap-api/api/transaction/v1"
)

type ITransactionV1 interface {
	ListTransactions(ctx context.Context, req *v1.ListTransactionsReq) (res *v1.ListTransactionsRes, err error)
	CreateTransaction(ctx context.Context, req *v1.CreateTransactionReq) (res *v1.CreateTransactionRes, err error)
	GetTransaction(ctx context.Context, req *v1.GetTransactionReq) (res *v1.GetTransactionRes, err error)
	UpdateTransaction(ctx context.Context, req *v1.UpdateTransactionReq) (res *v1.UpdateTransactionRes, err error)
	DeleteTransaction(ctx context.Context, req *v1.DeleteTransactionReq) (res *v1.DeleteTransactionRes, err error)
}
