package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfGetExchangeRates(ctx context.Context, req *v1.GfGetExchangeRatesReq) (res *v1.GfGetExchangeRatesRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetExchangeRatesReq); err != nil {
		return nil, err
	}

	snapshot, err := service.ExchangeRate().GetRates(ctx)
	if err != nil {
		return nil, err
	}

	rates := make([]*v1.ExchangeRate, 0, len(snapshot.Rates))
	for _, entry := range snapshot.Rates {
		rates = append(rates, &v1.ExchangeRate{
			Currency:  entry.Currency,
			Rate:      entry.Rate.String(),
			Source:    entry.Source,
			UpdatedAt: entry.UpdatedAt,
		})
	}

	return &v1.GetExchangeRatesRes{
		Anchor: snapshot.Anchor,
		Rates:  rates,
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
