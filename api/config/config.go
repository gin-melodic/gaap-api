// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package config

import (
	"context"

	"gaap-api/api/config/v1"
)

type IConfigV1 interface {
	GfListCurrencies(ctx context.Context, req *v1.GfListCurrenciesReq) (res *v1.GfListCurrenciesRes, err error)
	GfAddCurrency(ctx context.Context, req *v1.GfAddCurrencyReq) (res *v1.GfAddCurrencyRes, err error)
	GfDeleteCurrency(ctx context.Context, req *v1.GfDeleteCurrencyReq) (res *v1.GfDeleteCurrencyRes, err error)
	GfGetThemes(ctx context.Context, req *v1.GfGetThemesReq) (res *v1.GfGetThemesRes, err error)
	GfGetAccountTypes(ctx context.Context, req *v1.GfGetAccountTypesReq) (res *v1.GfGetAccountTypesRes, err error)
}
