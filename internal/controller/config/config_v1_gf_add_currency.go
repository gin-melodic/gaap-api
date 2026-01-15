package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfAddCurrency(ctx context.Context, req *v1.GfAddCurrencyReq) (res *v1.GfAddCurrencyRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.AddCurrencyReq); err != nil {
		return nil, err
	}

	currencies, err := service.Config().AddCurrency(ctx, req.GetCode())
	if err != nil {
		return nil, err
	}

	return &v1.AddCurrencyRes{
		Currencies: currencies,
		Base:       &base.BaseResponse{Message: "success"},
	}, nil
}
