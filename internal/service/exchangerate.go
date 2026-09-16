// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"

	"github.com/shopspring/decimal"
)

type (
	IExchangeRate interface {
		// GetRates returns every anchor-relative rate currently stored for the anchor
		// currency, ordered by quote-currency code. Rates are carried as exact decimals.
		GetRates(ctx context.Context) (*model.ExchangeRateSnapshot, error)
		// SetManualRate stores a manual override for the given currency, expressed as
		// "1 anchor = rate currency". Manual rates take precedence over reference rates
		// and are never overwritten by the daily reference sync.
		SetManualRate(ctx context.Context, currency string, rate string) (*model.ExchangeRateEntry, error)
		// UpsertReferenceRates stores reference rates for every tracked currency that
		// has a value in the supplied map. Existing manual overrides are preserved.
		UpsertReferenceRates(ctx context.Context, base string, rates map[string]decimal.Decimal) error
		// SyncOnce performs a single reference-rate sync from the configured provider
		// and refreshes every user's dashboard valuation.
		SyncOnce(ctx context.Context) error
		// StartScheduler starts the daily reference-rate sync loop. It returns
		// immediately; the first sync runs asynchronously.
		StartScheduler(ctx context.Context) error
		// Convert converts an exact amount from one currency to another using the
		// USD-anchored cross rate. With anchor A, rate(A→from)=Rf and rate(A→to)=Rt,
		// `amount` units of `from` equal `amount * Rt / Rf` units of `to`. The result
		// is rounded to 9 decimal places (nanos precision). Same-currency conversion
		// is the identity.
		Convert(ctx context.Context, amount decimal.Decimal, from string, to string) (decimal.Decimal, error)
		// MissingCurrencies reports which of the supplied currency codes lack a stored
		// anchor rate (the anchor itself is always considered complete).
		MissingCurrencies(ctx context.Context, used []string) ([]string, error)
	}
)

var (
	localExchangeRate IExchangeRate
)

func ExchangeRate() IExchangeRate {
	if localExchangeRate == nil {
		panic("implement not found for interface IExchangeRate, forgot register?")
	}
	return localExchangeRate
}

func RegisterExchangeRate(i IExchangeRate) {
	localExchangeRate = i
}
