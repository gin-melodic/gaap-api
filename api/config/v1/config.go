package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
)

type ListCurrenciesReq struct {
	g.Meta `path:"/config/currencies" tags:"Configuration" method:"get" summary:"Get supported currencies"`
}

type ListCurrenciesRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	Codes []string `json:"codes"`
}

type AddCurrencyReq struct {
	g.Meta `path:"/config/currencies" tags:"Configuration" method:"post" summary:"Add a supported currency"`
	Code   string `json:"code" v:"required"`
}

type AddCurrencyRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	Codes []string `json:"codes"`
}

type DeleteCurrencyReq struct {
	g.Meta `path:"/config/currencies" tags:"Configuration" method:"delete" summary:"Remove a supported currency"`
	Code   string `json:"code" v:"required"`
}

type DeleteCurrencyRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
}

type GetThemesReq struct {
	g.Meta `path:"/config/themes" tags:"Configuration" method:"get" summary:"Get available themes"`
}

type GetThemesRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	Themes []common.Theme `json:"themes"`
}

type GetAccountTypesReq struct {
	g.Meta `path:"/config/account-types" tags:"Configuration" method:"get" summary:"Get account type definitions"`
}

type GetAccountTypesRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	Types map[string]common.AccountTypeConfig `json:"types"`
}
