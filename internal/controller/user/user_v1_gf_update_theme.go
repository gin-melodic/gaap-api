package user

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/user/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfUpdateTheme(ctx context.Context, req *v1.GfUpdateThemeReq) (res *v1.GfUpdateThemeRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.UpdateThemePreferenceReq); err != nil {
		return nil, err
	}

	var themeId uuid.UUID
	if req.GetTheme() != nil && req.GetTheme().GetId() != "" {
		themeId, _ = uuid.Parse(req.GetTheme().GetId())
	}

	input := model.Theme{
		Id:     themeId,
		Name:   req.GetTheme().GetName(),
		IsDark: req.GetTheme().GetIsDark(),
	}

	if req.GetTheme().GetColors() != nil {
		input.Colors = model.ThemeColors{
			Primary: req.GetTheme().GetColors().GetPrimary(),
			Bg:      req.GetTheme().GetColors().GetBg(),
			Card:    req.GetTheme().GetColors().GetCard(),
			Text:    req.GetTheme().GetColors().GetText(),
			Muted:   req.GetTheme().GetColors().GetMuted(),
			Border:  req.GetTheme().GetColors().GetBorder(),
		}
	}

	theme, err := service.User().UpdateThemePreference(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.GfUpdateThemeRes{
		Theme: themeToProto(theme),
		Base:  &base.BaseResponse{Message: "success"},
	}, nil
}
