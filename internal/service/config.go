// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"
)

type (
	IConfig interface {
		ListCurrencies(ctx context.Context) (out []string, err error)
		AddCurrency(ctx context.Context, code string) (out []string, err error)
		DeleteCurrency(ctx context.Context, code string) (err error)
		// GetThemes returns all available themes
		GetThemes(ctx context.Context) (out []model.Theme, err error)
		// GetAccountTypes returns all account type configurations
		GetAccountTypes(ctx context.Context) (out map[int]model.AccountTypeConfig, err error)
	}
)

var (
	localConfig IConfig
)

func Config() IConfig {
	if localConfig == nil {
		panic("implement not found for interface IConfig, forgot register?")
	}
	return localConfig
}

func RegisterConfig(i IConfig) {
	localConfig = i
}
