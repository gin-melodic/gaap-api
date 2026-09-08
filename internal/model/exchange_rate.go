package model

import (
	"errors"

	"github.com/shopspring/decimal"
)

// ErrMissingRate is returned by the exchange-rate service when no rate exists
// for a currency needed by a valuation. Callers treat it as "incomplete" rather
// than a hard failure.
var ErrMissingRate = errors.New("missing exchange rate")

// ExchangeRateEntry is a single anchor-relative exchange rate for one currency.
// The rate expresses "1 anchor = rate units of Currency" and is carried as an
// exact decimal to satisfy the project's no-float rule for financial values.
type ExchangeRateEntry struct {
	Currency  string          `json:"currency"`
	Rate      decimal.Decimal `json:"rate"`
	Source    string          `json:"source"`
	UpdatedAt string          `json:"updatedAt"`
}

// ExchangeRateSnapshot is the full set of anchor-relative rates for a user
// together with the anchor currency code they are expressed against.
type ExchangeRateSnapshot struct {
	Anchor string              `json:"anchor"`
	Rates  []ExchangeRateEntry `json:"rates"`
}
