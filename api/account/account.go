package account

import (
	"context"
	v1 "gaap-api/api/account/v1"
)

type IAccountV1 interface {
	ListAccounts(ctx context.Context, req *v1.ListAccountsReq) (res *v1.ListAccountsRes, err error)
	CreateAccount(ctx context.Context, req *v1.CreateAccountReq) (res *v1.CreateAccountRes, err error)
	GetAccount(ctx context.Context, req *v1.GetAccountReq) (res *v1.GetAccountRes, err error)
	UpdateAccount(ctx context.Context, req *v1.UpdateAccountReq) (res *v1.UpdateAccountRes, err error)
	DeleteAccount(ctx context.Context, req *v1.DeleteAccountReq) (res *v1.DeleteAccountRes, err error)
}
