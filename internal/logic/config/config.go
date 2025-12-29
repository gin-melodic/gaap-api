package config

import (
	"context"
	"encoding/json"

	"gaap-api/internal/dao"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"
)

type sConfig struct{}

func init() {
	service.RegisterConfig(New())
}

func New() *sConfig {
	return &sConfig{}
}

func (s *sConfig) ListCurrencies(ctx context.Context) (out []string, err error) {
	var results []struct {
		Code string
	}
	err = dao.Currencies.Ctx(ctx).Fields("code").WhereNull("deleted_at").Scan(&results)
	if err != nil {
		return
	}
	for _, r := range results {
		out = append(out, r.Code)
	}
	return
}

func (s *sConfig) AddCurrency(ctx context.Context, code string) (out []string, err error) {
	_, err = dao.Currencies.Ctx(ctx).Data(map[string]interface{}{"code": code}).Insert()
	if err != nil {
		return
	}
	return s.ListCurrencies(ctx)
}

func (s *sConfig) DeleteCurrency(ctx context.Context, code string) (err error) {
	_, err = dao.Currencies.Ctx(ctx).Unscoped().Where("code", code).Delete()
	return
}

// GetThemes returns all available themes
func (s *sConfig) GetThemes(ctx context.Context) (out []model.Theme, err error) {
	var themes []entity.Themes
	err = dao.Themes.Ctx(ctx).WhereNull("deleted_at").Scan(&themes)
	if err != nil {
		return
	}

	out = make([]model.Theme, 0, len(themes))
	for _, t := range themes {
		theme := model.Theme{
			Id:     t.Id,
			Name:   t.Name,
			IsDark: t.IsDark,
		}

		// Parse JSONB colors field
		if t.Colors != "" {
			if err := json.Unmarshal([]byte(t.Colors), &theme.Colors); err != nil {
				// If parsing fails, skip colors but don't fail the whole request
				theme.Colors = model.ThemeColors{}
			}
		}

		out = append(out, theme)
	}
	return
}

// GetAccountTypes returns all account type configurations
func (s *sConfig) GetAccountTypes(ctx context.Context) (out map[string]model.AccountTypeConfig, err error) {
	var accountTypes []entity.AccountTypes
	err = dao.AccountTypes.Ctx(ctx).WhereNull("deleted_at").Scan(&accountTypes)
	if err != nil {
		return
	}

	out = make(map[string]model.AccountTypeConfig)
	for _, at := range accountTypes {
		out[at.Type] = model.AccountTypeConfig{
			Label: at.Label,
			Color: at.Color,
			Bg:    at.Bg,
			Icon:  at.Icon,
		}
	}
	return
}
