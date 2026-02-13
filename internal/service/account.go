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
	IAccount interface {
		ListAccounts(ctx context.Context, in model.AccountQueryInput) (out []entity.Accounts, total int, err error)
		CreateAccount(ctx context.Context, in model.AccountCreateInput) (out *entity.Accounts, err error)
		GetAccount(ctx context.Context, id uuid.UUID) (out *entity.Accounts, err error)
		UpdateAccount(ctx context.Context, id uuid.UUID, in model.AccountUpdateInput) (out *entity.Accounts, err error)
		DeleteAccount(ctx context.Context, id uuid.UUID, migrationTargets map[string]uuid.UUID) (taskId string, err error)
		// GetAccountTransactionCount returns the number of transactions involving this account, and the number of transactions involving this account without equity
		GetAccountTransactionCount(ctx context.Context, id uuid.UUID) (count int, countWithoutEquity int, err error)
	}
)

var (
	localAccount IAccount
)

func Account() IAccount {
	if localAccount == nil {
		panic("implement not found for interface IAccount, forgot register?")
	}
	return localAccount
}

func RegisterAccount(i IAccount) {
	localAccount = i
}
