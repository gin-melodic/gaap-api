package auth

import (
	"context"

	"gaap-api/api/auth/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfGetCurrencyList(ctx context.Context, req *v1.GfGetCurrencyListReq) (res *v1.GfGetCurrencyListRes, err error) {
	codes, err := service.Auth().GetCurrencyList(ctx)
	if err != nil {
		return nil, err
	}

	res = &v1.GfGetCurrencyListRes{}
	for _, code := range codes {
		res.Currencies = append(res.Currencies, &v1.CurrencyInfo{
			Code: code,
		})
	}
	return res, nil
}
