package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfDeleteCurrency(ctx context.Context, req *v1.GfDeleteCurrencyReq) (res *v1.GfDeleteCurrencyRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.DeleteCurrencyReq); err != nil {
		return nil, err
	}

	err = service.Config().DeleteCurrency(ctx, req.GetCode())
	if err != nil {
		return nil, err
	}

	return &v1.DeleteCurrencyRes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
