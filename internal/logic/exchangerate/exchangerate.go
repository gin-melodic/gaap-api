package exchangerate

import (
	"context"
	"strings"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/shopspring/decimal"
)

const (
	rateSyncInterval = 24 * time.Hour
	rateSourceManual = "manual"
	rateSourceRef    = "reference"
)

type sExchangeRate struct {
	provider ExchangeRateProvider
}

func init() {
	service.RegisterExchangeRate(New())
}

// New constructs the exchange-rate service with the default fawazahmed0 provider.
func New() *sExchangeRate {
	return &sExchangeRate{provider: NewFawazahmed0Provider()}
}

// GetRates returns every anchor-relative rate currently stored for the anchor
// currency, ordered by quote-currency code. Rates are carried as exact decimals.
func (s *sExchangeRate) GetRates(ctx context.Context) (*model.ExchangeRateSnapshot, error) {
	anchor := AnchorCurrency()
	rows, err := s.loadRates(ctx, anchor)
	if err != nil {
		return nil, err
	}
	return &model.ExchangeRateSnapshot{Anchor: anchor, Rates: rows}, nil
}

// SetManualRate stores a manual override for the given currency, expressed as
// "1 anchor = rate currency". Manual rates take precedence over reference rates
// and are never overwritten by the daily reference sync.
func (s *sExchangeRate) SetManualRate(ctx context.Context, currency string, rate string) (*model.ExchangeRateEntry, error) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if len(code) != 3 {
		return nil, gerror.New("currency code must be a 3-letter ISO code")
	}
	anchor := AnchorCurrency()
	if code == anchor {
		return nil, gerror.New("anchor currency rate is fixed at 1")
	}

	value, err := decimal.NewFromString(strings.TrimSpace(rate))
	if err != nil || !value.IsPositive() {
		return nil, gerror.New("rate must be a positive decimal")
	}

	if err := s.ensureTrackedCurrency(ctx, code); err != nil {
		return nil, err
	}

	if err := s.upsertRate(ctx, anchor, code, value, rateSourceManual); err != nil {
		return nil, err
	}

	go s.refreshAllUsers(context.Background())

	return &model.ExchangeRateEntry{
		Currency:  code,
		Rate:      value,
		Source:    rateSourceManual,
		UpdatedAt: gtime.Now().String(),
	}, nil
}

// UpsertReferenceRates stores reference rates for every tracked currency that
// has a value in the supplied map. Existing manual overrides are preserved.
func (s *sExchangeRate) UpsertReferenceRates(ctx context.Context, base string, rates map[string]decimal.Decimal) error {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		return gerror.New("reference rate base currency is required")
	}

	tracked, err := s.trackedCurrencies(ctx)
	if err != nil {
		return err
	}
	manualRows, err := s.manualRows(ctx, base)
	if err != nil {
		return err
	}

	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		for _, code := range tracked {
			if code == base {
				continue
			}
			if _, isManual := manualRows[code]; isManual {
				continue
			}
			value, ok := rates[code]
			if !ok || !value.IsPositive() {
				continue
			}
			if err := s.upsertRateInTx(ctx, tx, base, code, value, rateSourceRef); err != nil {
				return err
			}
		}
		return nil
	})
}

// SyncOnce performs a single reference-rate sync from the configured provider
// and refreshes every user's dashboard valuation.
func (s *sExchangeRate) SyncOnce(ctx context.Context) error {
	if !syncEnabled() {
		return nil
	}
	anchor := AnchorCurrency()
	rates, err := s.provider.FetchRates(ctx, anchor)
	if err != nil {
		return gerror.Wrap(err, "exchange rate sync failed")
	}
	if err := s.UpsertReferenceRates(ctx, anchor, rates); err != nil {
		return gerror.Wrap(err, "failed to persist reference rates")
	}
	s.refreshAllUsers(ctx)
	return nil
}

// StartScheduler starts the daily reference-rate sync loop. It returns
// immediately; the first sync runs asynchronously.
func (s *sExchangeRate) StartScheduler(ctx context.Context) error {
	if !syncEnabled() {
		g.Log().Info(ctx, "exchange rate sync is disabled; scheduler not started")
		return nil
	}
	go s.runScheduler(ctx)
	return nil
}

func (s *sExchangeRate) runScheduler(ctx context.Context) {
	s.syncWithLog(ctx)
	ticker := time.NewTicker(rateSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncWithLog(ctx)
		}
	}
}

func (s *sExchangeRate) syncWithLog(ctx context.Context) {
	if err := s.SyncOnce(ctx); err != nil {
		g.Log().Warningf(ctx, "exchange rate sync failed: %v", err)
	}
}

// upsertRate performs an atomic upsert of a single (base, quote) rate row.
func (s *sExchangeRate) upsertRate(ctx context.Context, base, quote string, rate decimal.Decimal, source string) error {
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		return s.upsertRateInTx(ctx, tx, base, quote, rate, source)
	})
}

func (s *sExchangeRate) upsertRateInTx(ctx context.Context, tx gdb.TX, base, quote string, rate decimal.Decimal, source string) error {
	now := gtime.Now()
	_, err := tx.Model(dao.ExchangeRates.Table()).
		Data(g.Map{
			dao.ExchangeRates.Columns().BaseCurrency:  base,
			dao.ExchangeRates.Columns().QuoteCurrency: quote,
			dao.ExchangeRates.Columns().Rate:          rate.String(),
			dao.ExchangeRates.Columns().Source:        source,
			dao.ExchangeRates.Columns().FetchedAt:     now,
			dao.ExchangeRates.Columns().UpdatedAt:     now,
		}).
		OnConflict(dao.ExchangeRates.Columns().BaseCurrency, dao.ExchangeRates.Columns().QuoteCurrency).
		Save()
	if err != nil {
		return gerror.Wrap(err, "failed to upsert exchange rate")
	}
	return nil
}

// loadRates reads all stored rates for a base currency as exact decimals.
func (s *sExchangeRate) loadRates(ctx context.Context, base string) ([]model.ExchangeRateEntry, error) {
	var rows []rateRow
	err := dao.ExchangeRates.Ctx(ctx).
		Fields(
			dao.ExchangeRates.Columns().QuoteCurrency,
			"rate::text AS rate_text",
			dao.ExchangeRates.Columns().Source,
			dao.ExchangeRates.Columns().UpdatedAt,
		).
		Where(dao.ExchangeRates.Columns().BaseCurrency, base).
		OrderAsc(dao.ExchangeRates.Columns().QuoteCurrency).
		Scan(&rows)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to load exchange rates")
	}

	out := make([]model.ExchangeRateEntry, 0, len(rows))
	for _, row := range rows {
		value, err := decimal.NewFromString(row.RateText)
		if err != nil {
			return nil, gerror.Wrapf(err, "invalid stored exchange rate for %q", row.QuoteCurrency)
		}
		updatedAt := ""
		if row.UpdatedAt != nil {
			updatedAt = row.UpdatedAt.String()
		}
		out = append(out, model.ExchangeRateEntry{
			Currency:  row.QuoteCurrency,
			Rate:      value,
			Source:    row.Source,
			UpdatedAt: updatedAt,
		})
	}
	return out, nil
}

// manualRows returns the set of quote-currency codes that currently hold a
// manual override for the given base currency.
func (s *sExchangeRate) manualRows(ctx context.Context, base string) (map[string]struct{}, error) {
	var rows []struct {
		QuoteCurrency string `orm:"quote_currency"`
	}
	err := dao.ExchangeRates.Ctx(ctx).
		Fields(dao.ExchangeRates.Columns().QuoteCurrency).
		Where(dao.ExchangeRates.Columns().BaseCurrency, base).
		Where(dao.ExchangeRates.Columns().Source, rateSourceManual).
		Scan(&rows)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to load manual exchange rates")
	}
	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[row.QuoteCurrency] = struct{}{}
	}
	return out, nil
}

// trackedCurrencies returns the uppercase codes of all supported currencies.
func (s *sExchangeRate) trackedCurrencies(ctx context.Context) ([]string, error) {
	var rows []entity.Currencies
	err := dao.Currencies.Ctx(ctx).
		Fields(dao.Currencies.Columns().Code).
		WhereNull(dao.Currencies.Columns().DeletedAt).
		Scan(&rows)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to load supported currencies")
	}
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, strings.ToUpper(strings.TrimSpace(row.Code)))
	}
	return codes, nil
}

// ensureTrackedCurrency returns an error unless the code is a supported currency.
func (s *sExchangeRate) ensureTrackedCurrency(ctx context.Context, code string) error {
	count, err := dao.Currencies.Ctx(ctx).
		Where(dao.Currencies.Columns().Code, code).
		WhereNull(dao.Currencies.Columns().DeletedAt).
		Count()
	if err != nil {
		return gerror.Wrap(err, "failed to validate currency")
	}
	if count == 0 {
		return gerror.Newf("unsupported currency %q", code)
	}
	return nil
}

// refreshAllUsers invalidates and republishes dashboard snapshots for every
// active user so valuations reflect the new rates.
func (s *sExchangeRate) refreshAllUsers(ctx context.Context) {
	var users []struct {
		Id string `orm:"id"`
	}
	err := dao.Users.Ctx(ctx).
		Fields(dao.Users.Columns().Id).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&users)
	if err != nil {
		g.Log().Warningf(ctx, "failed to list users for rate refresh: %v", err)
		return
	}
	for _, user := range users {
		dashboard.PublishDashboardRefresh(ctx, user.Id, "rate_sync")
	}
}

// rateRow is the scan target for exchange-rate rows; RateText keeps the NUMERIC
// value as text so no floating-point conversion occurs.
type rateRow struct {
	QuoteCurrency string      `orm:"quote_currency"`
	RateText      string      `orm:"rate_text"`
	Source        string      `orm:"source"`
	UpdatedAt     *gtime.Time `orm:"updated_at"`
}
