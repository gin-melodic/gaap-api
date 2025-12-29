package user

import (
	"context"
	common "gaap-api/api/common/v1"
	v1 "gaap-api/api/user/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
)

func (c *ControllerV1) GetUserProfile(ctx context.Context, req *v1.GetUserProfileReq) (res *v1.GetUserProfileRes, err error) {
	out, err := service.User().GetUserProfile(ctx)
	if err != nil {
		return &v1.GetUserProfileRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.GetUserProfileRes{
		User: &v1.User{
			Email:            out.Email,
			Nickname:         out.Nickname,
			Avatar:           &out.Avatar,
			Plan:             out.Plan,
			TwoFactorEnabled: out.TwoFactorEnabled,
			MainCurrency:     out.MainCurrency,
		},
	}
	return
}

func (c *ControllerV1) UpdateUserProfile(ctx context.Context, req *v1.UpdateUserProfileReq) (res *v1.UpdateUserProfileRes, err error) {
	in := model.UserUpdateInput{
		Nickname: req.Nickname,
		Plan:     req.Plan,
	}
	if req.Avatar != nil {
		in.Avatar = *req.Avatar
	}
	out, err := service.User().UpdateUserProfile(ctx, in)
	if err != nil {
		return &v1.UpdateUserProfileRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.UpdateUserProfileRes{
		User: &v1.User{
			Email:            out.Email,
			Nickname:         out.Nickname,
			Avatar:           &out.Avatar,
			Plan:             out.Plan,
			TwoFactorEnabled: out.TwoFactorEnabled,
			MainCurrency:     out.MainCurrency,
		},
	}
	return
}

func (c *ControllerV1) UpdateThemePreference(ctx context.Context, req *v1.UpdateThemePreferenceReq) (res *v1.UpdateThemePreferenceRes, err error) {
	in := model.Theme{
		Id:     req.Id,
		Name:   req.Name,
		IsDark: req.IsDark,
	}
	if req.Colors != nil {
		in.Colors = model.ThemeColors{
			Primary: req.Colors.Primary,
			Bg:      req.Colors.Bg,
			Card:    req.Colors.Card,
			Text:    req.Colors.Text,
			Muted:   req.Colors.Muted,
			Border:  req.Colors.Border,
		}
	}

	out, err := service.User().UpdateThemePreference(ctx, in)
	if err != nil {
		return &v1.UpdateThemePreferenceRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil // Should not happen if err is nil
	}

	res = &v1.UpdateThemePreferenceRes{
		Theme: &common.Theme{
			Id:     out.Id,
			Name:   out.Name,
			IsDark: out.IsDark,
			Colors: &common.ThemeColors{
				Primary: out.Colors.Primary,
				Bg:      out.Colors.Bg,
				Card:    out.Colors.Card,
				Text:    out.Colors.Text,
				Muted:   out.Colors.Muted,
				Border:  out.Colors.Border,
			},
		},
	}
	return
}
