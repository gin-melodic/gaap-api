package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfSetExchangeRate(ctx context.Context, req *v1.GfSetExchangeRateReq) (res *v1.GfSetExchangeRateRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.SetExchangeRateReq); err != nil {
		return nil, err
	}

	entry, err := service.ExchangeRate().SetManualRate(ctx, req.GetCurrency(), req.GetRate())
	if err != nil {
		return nil, err
	}

	return &v1.SetExchangeRateRes{
		Rate: &v1.ExchangeRate{
			Currency:  entry.Currency,
			Rate:      entry.Rate.String(),
			Source:    entry.Source,
			UpdatedAt: entry.UpdatedAt,
		},
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
