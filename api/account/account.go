// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package account

import (
	"context"

	"gaap-api/api/account/v1"
)

type IAccountV1 interface {
	GfListAccounts(ctx context.Context, req *v1.GfListAccountsReq) (res *v1.GfListAccountsRes, err error)
	GfCreateAccount(ctx context.Context, req *v1.GfCreateAccountReq) (res *v1.GfCreateAccountRes, err error)
	GfGetAccount(ctx context.Context, req *v1.GfGetAccountReq) (res *v1.GfGetAccountRes, err error)
	GfUpdateAccount(ctx context.Context, req *v1.GfUpdateAccountReq) (res *v1.GfUpdateAccountRes, err error)
	GfDeleteAccount(ctx context.Context, req *v1.GfDeleteAccountReq) (res *v1.GfDeleteAccountRes, err error)
	GfGetAccountTransactionCount(ctx context.Context, req *v1.GfGetAccountTransactionCountReq) (res *v1.GfGetAccountTransactionCountRes, err error)
}
