package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GfListCurrencies(ctx context.Context, req *v1.GfListCurrenciesReq) (res *v1.GfListCurrenciesRes, err error) {
	currencies, err := service.Config().ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListCurrenciesRes{
		Currencies: currencies,
		Base:       &base.BaseResponse{Message: "success"},
	}, nil
}
