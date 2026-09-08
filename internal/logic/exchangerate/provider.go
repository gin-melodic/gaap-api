package exchangerate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/shopspring/decimal"
)

const (
	defaultProviderBaseURL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1"
	defaultAnchorCurrency  = "USD"
	providerTimeout        = 15 * time.Second
)

// ExchangeRateProvider abstracts a source of daily reference exchange rates.
// Implementations return a map of quote-currency codes (uppercase) to the
// amount of that currency per one unit of the given base currency.
type ExchangeRateProvider interface {
	FetchRates(ctx context.Context, base string) (map[string]decimal.Decimal, error)
}

// fawazahmed0Provider fetches rates from the fawazahmed0/currency-api CDN.
type fawazahmed0Provider struct {
	baseURL string
	client  *http.Client
}

// NewFawazahmed0Provider constructs the default reference-rate provider.
func NewFawazahmed0Provider() ExchangeRateProvider {
	return &fawazahmed0Provider{
		baseURL: strings.TrimRight(providerBaseURL(), "/"),
		client:  &http.Client{Timeout: providerTimeout},
	}
}

// providerBaseURL returns the configured provider base URL, falling back to the
// fawazahmed0 CDN already used by the frontend.
func providerBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("EXCHANGE_RATE_PROVIDER_URL")); v != "" {
		return v
	}
	return defaultProviderBaseURL
}

// AnchorCurrency returns the configured anchor currency code (uppercase).
func AnchorCurrency() string {
	v := strings.ToUpper(strings.TrimSpace(os.Getenv("EXCHANGE_RATE_ANCHOR")))
	if v == "" {
		return defaultAnchorCurrency
	}
	return v
}

// syncEnabled reports whether the daily reference-rate sync should run.
func syncEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("EXCHANGE_RATE_SYNC_ENABLED")))
	if v == "" {
		return true
	}
	return v == "true" || v == "1" || v == "yes"
}

// FetchRates returns rates relative to the given base currency, normalizing
// every quote-currency code to uppercase. Numbers are decoded with UseNumber so
// no floating-point conversion occurs.
func (p *fawazahmed0Provider) FetchRates(ctx context.Context, base string) (map[string]decimal.Decimal, error) {
	base = strings.ToLower(strings.TrimSpace(base))
	if base == "" {
		return nil, gerror.New("exchange rate base currency is required")
	}

	url := fmt.Sprintf("%s/currencies/%s.json", p.baseURL, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to build exchange rate request")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, gerror.Wrap(err, "exchange rate provider request failed")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, gerror.Newf("exchange rate provider returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, gerror.Wrap(err, "failed to read exchange rate response")
	}

	return parseFawazahmed0Response(body, base)
}

// parseFawazahmed0Response parses the JSON shape produced by fawazahmed0:
//
//	{ "date": "...", "usd": { "cny": 7.2, "jpy": 150.0, ... } }
func parseFawazahmed0Response(body []byte, base string) (map[string]decimal.Decimal, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var envelope map[string]json.RawMessage
	if err := decoder.Decode(&envelope); err != nil {
		return nil, gerror.Wrap(err, "failed to decode exchange rate response")
	}

	baseKey := strings.ToLower(base)
	rawRates, ok := envelope[baseKey]
	if !ok {
		return nil, gerror.Newf("exchange rate response missing base %q", base)
	}

	var quoteMap map[string]json.Number
	quoteDecoder := json.NewDecoder(bytes.NewReader(rawRates))
	quoteDecoder.UseNumber()
	if err := quoteDecoder.Decode(&quoteMap); err != nil {
		return nil, gerror.Wrap(err, "failed to decode exchange rate quotes")
	}

	rates := make(map[string]decimal.Decimal, len(quoteMap)+1)
	rates[strings.ToUpper(base)] = decimal.NewFromInt(1)
	for code, number := range quoteMap {
		value, err := decimal.NewFromString(number.String())
		if err != nil {
			return nil, gerror.Wrapf(err, "invalid exchange rate for %q", code)
		}
		rates[strings.ToUpper(code)] = value
	}
	return rates, nil
}
