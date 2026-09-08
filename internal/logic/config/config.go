package config

import (
	"context"
	"encoding/json"
	"strings"

	"gaap-api/internal/dao"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type sConfig struct{}

func init() {
	service.RegisterConfig(New())
}

func New() *sConfig {
	return &sConfig{}
}

func (s *sConfig) ListCurrencies(ctx context.Context) (out []string, err error) {
	var results []entity.Currencies
	err = dao.Currencies.Ctx(ctx).
		Fields(dao.Currencies.Columns().Code).
		WhereNull(dao.Currencies.Columns().DeletedAt).
		Scan(&results)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to list currencies")
	}
	for _, r := range results {
		out = append(out, r.Code)
	}
	return
}

func (s *sConfig) AddCurrency(ctx context.Context, code string) (out []string, err error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 3 {
		return nil, gerror.New("currency code must be a 3-letter ISO code")
	}
	currency := entity.Currencies{
		Code: code,
	}
	_, err = dao.Currencies.Ctx(ctx).Data(currency).Insert()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to add currency")
	}
	return s.ListCurrencies(ctx)
}

func (s *sConfig) DeleteCurrency(ctx context.Context, code string) (err error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return gerror.New("currency code is required")
	}

	inUse, err := dao.Accounts.Ctx(ctx).
		Where(dao.Accounts.Columns().CurrencyCode, code).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		Count()
	if err != nil {
		return gerror.Wrap(err, "failed to check currency usage")
	}
	if inUse > 0 {
		return gerror.New("currency is in use by accounts")
	}

	now := gtime.Now()
	_, err = dao.Currencies.Ctx(ctx).
		Where(dao.Currencies.Columns().Code, code).
		Data(g.Map{
			dao.Currencies.Columns().DeletedAt: now,
			dao.Currencies.Columns().UpdatedAt: now,
		}).
		Update()
	if err != nil {
		return gerror.Wrap(err, "failed to delete currency")
	}
	return
}

// GetThemes returns all available themes
func (s *sConfig) GetThemes(ctx context.Context) (out []model.Theme, err error) {
	var themes []entity.Themes
	err = dao.Themes.Ctx(ctx).WhereNull(dao.Themes.Columns().DeletedAt).Scan(&themes)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get themes")
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
func (s *sConfig) GetAccountTypes(ctx context.Context) (out map[int]model.AccountTypeConfig, err error) {
	var accountTypes []entity.AccountTypes
	err = dao.AccountTypes.Ctx(ctx).WhereNull(dao.AccountTypes.Columns().DeletedAt).Scan(&accountTypes)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get account types")
	}

	out = make(map[int]model.AccountTypeConfig)
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
