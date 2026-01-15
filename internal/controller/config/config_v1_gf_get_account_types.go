package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfGetAccountTypes(ctx context.Context, req *v1.GfGetAccountTypesReq) (res *v1.GfGetAccountTypesRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetAccountTypesReq); err != nil {
		return nil, err
	}

	accountTypes, err := service.Config().GetAccountTypes(ctx)
	if err != nil {
		return nil, err
	}

	// Convert map[int]model.AccountTypeConfig to map[int32]*base.AccountTypeConfig
	result := make(map[int32]*base.AccountTypeConfig, len(accountTypes))
	for k, v := range accountTypes {
		result[int32(k)] = accountTypeConfigToProto(v)
	}

	return &v1.GetAccountTypesRes{
		Types: result,
		Base:  &base.BaseResponse{Message: "success"},
	}, nil
}
