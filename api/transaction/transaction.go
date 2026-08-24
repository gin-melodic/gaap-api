// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package transaction

import (
	"context"

	"gaap-api/api/transaction/v1"
)

type ITransactionV1 interface {
	GfListTransactions(ctx context.Context, req *v1.GfListTransactionsReq) (res *v1.GfListTransactionsRes, err error)
	GfCreateTransaction(ctx context.Context, req *v1.GfCreateTransactionReq) (res *v1.GfCreateTransactionRes, err error)
	GfGetTransaction(ctx context.Context, req *v1.GfGetTransactionReq) (res *v1.GfGetTransactionRes, err error)
	GfUpdateTransaction(ctx context.Context, req *v1.GfUpdateTransactionReq) (res *v1.GfUpdateTransactionRes, err error)
	GfDeleteTransaction(ctx context.Context, req *v1.GfDeleteTransactionReq) (res *v1.GfDeleteTransactionRes, err error)
}
