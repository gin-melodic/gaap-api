package config

import (
	"context"
	common "gaap-api/api/common/v1"
	v1 "gaap-api/api/config/v1"
	"gaap-api/internal/service"
)

func (c *ControllerV1) ListCurrencies(ctx context.Context, req *v1.ListCurrenciesReq) (res *v1.ListCurrenciesRes, err error) {
	out, err := service.Config().ListCurrencies(ctx)
	if err != nil {
		return &v1.ListCurrenciesRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.ListCurrenciesRes{
		Codes: out,
	}
	return
}

func (c *ControllerV1) AddCurrency(ctx context.Context, req *v1.AddCurrencyReq) (res *v1.AddCurrencyRes, err error) {
	out, err := service.Config().AddCurrency(ctx, req.Code)
	if err != nil {
		return &v1.AddCurrencyRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.AddCurrencyRes{
		Codes: out,
	}
	return
}

func (c *ControllerV1) DeleteCurrency(ctx context.Context, req *v1.DeleteCurrencyReq) (res *v1.DeleteCurrencyRes, err error) {
	err = service.Config().DeleteCurrency(ctx, req.Code)
	if err != nil {
		return &v1.DeleteCurrencyRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.DeleteCurrencyRes{}
	return
}

func (c *ControllerV1) GetThemes(ctx context.Context, req *v1.GetThemesReq) (res *v1.GetThemesRes, err error) {
	out, err := service.Config().GetThemes(ctx)
	if err != nil {
		return &v1.GetThemesRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}

	themes := make([]common.Theme, 0, len(out))
	for _, t := range out {
		themes = append(themes, common.Theme{
			Id:     t.Id,
			Name:   t.Name,
			IsDark: t.IsDark,
			Colors: &common.ThemeColors{
				Primary: t.Colors.Primary,
				Bg:      t.Colors.Bg,
				Card:    t.Colors.Card,
				Text:    t.Colors.Text,
				Muted:   t.Colors.Muted,
				Border:  t.Colors.Border,
			},
		})
	}

	res = &v1.GetThemesRes{
		Themes: themes,
	}
	return
}

func (c *ControllerV1) GetAccountTypes(ctx context.Context, req *v1.GetAccountTypesReq) (res *v1.GetAccountTypesRes, err error) {
	out, err := service.Config().GetAccountTypes(ctx)
	if err != nil {
		return &v1.GetAccountTypesRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}

	types := make(map[string]common.AccountTypeConfig)
	for k, v := range out {
		types[k] = common.AccountTypeConfig{
			Label: v.Label,
			Color: v.Color,
			Bg:    v.Bg,
			Icon:  v.Icon,
		}
	}

	res = &v1.GetAccountTypesRes{
		Types: types,
	}
	return
}
