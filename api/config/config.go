package config

import (
	"context"
	v1 "gaap-api/api/config/v1"
)

type IConfigV1 interface {
	ListCurrencies(ctx context.Context, req *v1.ListCurrenciesReq) (res *v1.ListCurrenciesRes, err error)
	AddCurrency(ctx context.Context, req *v1.AddCurrencyReq) (res *v1.AddCurrencyRes, err error)
	DeleteCurrency(ctx context.Context, req *v1.DeleteCurrencyReq) (res *v1.DeleteCurrencyRes, err error)
	GetThemes(ctx context.Context, req *v1.GetThemesReq) (res *v1.GetThemesRes, err error)
	GetAccountTypes(ctx context.Context, req *v1.GetAccountTypesReq) (res *v1.GetAccountTypesRes, err error)
}
