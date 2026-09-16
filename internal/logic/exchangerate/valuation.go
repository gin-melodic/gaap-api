package exchangerate

import (
	"context"
	"strings"

	"gaap-api/internal/dao"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/shopspring/decimal"
)

// Convert converts an exact amount from one currency to another using the
// USD-anchored cross rate. With anchor A, rate(A→from)=Rf and rate(A→to)=Rt,
// `amount` units of `from` equal `amount * Rt / Rf` units of `to`. The result
// is rounded to 9 decimal places (nanos precision). Same-currency conversion
// is the identity.
func (s *sExchangeRate) Convert(ctx context.Context, amount decimal.Decimal, from string, to string) (decimal.Decimal, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == to {
		return amount, nil
	}

	anchor := AnchorCurrency()
	rateFrom, ok, err := s.loadRate(ctx, anchor, from)
	if err != nil {
		return decimal.Zero, err
	}
	if !ok {
		return decimal.Zero, model.ErrMissingRate
	}
	rateTo, ok, err := s.loadRate(ctx, anchor, to)
	if err != nil {
		return decimal.Zero, err
	}
	if !ok {
		return decimal.Zero, model.ErrMissingRate
	}

	return convertWithRates(amount, rateFrom, rateTo), nil
}

// convertWithRates converts an amount from one currency to another given the
// two anchor rates: rateFrom ("1 anchor = rateFrom from") and rateTo
// ("1 anchor = rateTo to"). The result is rounded to 9 decimal places.
func convertWithRates(amount decimal.Decimal, rateFrom decimal.Decimal, rateTo decimal.Decimal) decimal.Decimal {
	return amount.Mul(rateTo).DivRound(rateFrom, 9)
}

// MissingCurrencies reports which of the supplied currency codes lack a stored
// anchor rate (the anchor itself is always considered complete).
func (s *sExchangeRate) MissingCurrencies(ctx context.Context, used []string) ([]string, error) {
	anchor := AnchorCurrency()
	seen := make(map[string]struct{}, len(used))
	missing := make([]string, 0)
	for _, code := range used {
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" || code == anchor {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}

		_, ok, err := s.loadRate(ctx, anchor, code)
		if err != nil {
			return nil, err
		}
		if !ok {
			missing = append(missing, code)
		}
	}
	return missing, nil
}

// loadRate returns the anchor rate for a single quote currency. The anchor
// currency itself resolves to exactly 1 without a stored row.
func (s *sExchangeRate) loadRate(ctx context.Context, anchor string, quote string) (decimal.Decimal, bool, error) {
	if strings.EqualFold(anchor, quote) {
		return decimal.NewFromInt(1), true, nil
	}

	var row struct {
		RateText string `orm:"rate_text"`
	}
	err := dao.ExchangeRates.Ctx(ctx).
		Fields("rate::text AS rate_text").
		Where(dao.ExchangeRates.Columns().BaseCurrency, anchor).
		Where(dao.ExchangeRates.Columns().QuoteCurrency, quote).
		Scan(&row)
	if err != nil {
		return decimal.Zero, false, gerror.Wrap(err, "failed to load exchange rate")
	}
	if row.RateText == "" {
		return decimal.Zero, false, nil
	}
	value, err := decimal.NewFromString(row.RateText)
	if err != nil {
		return decimal.Zero, false, gerror.Wrap(err, "invalid stored exchange rate")
	}
	return value, true, nil
}
