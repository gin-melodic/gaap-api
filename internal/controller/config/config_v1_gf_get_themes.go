package config

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfGetThemes(ctx context.Context, req *v1.GfGetThemesReq) (res *v1.GfGetThemesRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetThemesReq); err != nil {
		return nil, err
	}

	themes, err := service.Config().GetThemes(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GetThemesRes{
		Themes: themesToProtos(themes),
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
