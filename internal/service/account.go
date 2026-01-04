package service

import (
	"context"
	"gaap-api/internal/model"
)

type IAccount interface {
	ListAccounts(ctx context.Context, in model.AccountQueryInput) (out []model.Account, total int, err error)
	CreateAccount(ctx context.Context, in model.AccountCreateInput) (out *model.Account, err error)
	GetAccount(ctx context.Context, id string) (out *model.Account, err error)
	UpdateAccount(ctx context.Context, id string, in model.AccountUpdateInput) (out *model.Account, err error)
	DeleteAccount(ctx context.Context, id string, migrationTargets map[string]string) (taskId string, err error)
	GetAccountTransactionCount(ctx context.Context, id string) (count int, err error)
}

var localAccount IAccount

func Account() IAccount {
	if localAccount == nil {
		panic("implement not found for interface IAccount, forgot register?")
	}
	return localAccount
}

func RegisterAccount(i IAccount) {
	localAccount = i
}
