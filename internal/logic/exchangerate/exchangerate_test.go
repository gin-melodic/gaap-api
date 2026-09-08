package exchangerate

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestConvertWithRates(t *testing.T) {
	t.Parallel()

	t.Run("same anchor identity", func(t *testing.T) {
		got := convertWithRates(
			decimal.RequireFromString("123.45"),
			decimal.NewFromInt(1),
			decimal.NewFromInt(1),
		)
		require.True(t, got.Equal(decimal.RequireFromString("123.45")), "identity conversion changed the amount")
	})

	t.Run("cn to usd", func(t *testing.T) {
		// 1 USD = 7.2 CNY, so 100 CNY = 100 / 7.2 USD = 13.888888888...
		got := convertWithRates(
			decimal.NewFromInt(100),
			decimal.RequireFromString("7.2"),
			decimal.NewFromInt(1),
		)
		require.Equal(t, "13.888888889", got.String())
	})

	t.Run("usd to cn", func(t *testing.T) {
		got := convertWithRates(
			decimal.NewFromInt(100),
			decimal.NewFromInt(1),
			decimal.RequireFromString("7.2"),
		)
		require.Equal(t, "720", got.String())
	})

	t.Run("cross currency jpy to eur", func(t *testing.T) {
		// 1 USD = 150 JPY and 1 USD = 0.92 EUR.
		// 15000 JPY = 15000 * 0.92 / 150 EUR = 92 EUR.
		got := convertWithRates(
			decimal.NewFromInt(15000),
			decimal.NewFromInt(150),
			decimal.RequireFromString("0.92"),
		)
		require.Equal(t, "92", got.String())
	})

	t.Run("rounds to nine decimal places", func(t *testing.T) {
		got := convertWithRates(
			decimal.NewFromInt(1),
			decimal.NewFromInt(3),
			decimal.NewFromInt(1),
		)
		require.Equal(t, "0.333333333", got.String())
	})

	t.Run("zero amount", func(t *testing.T) {
		got := convertWithRates(
			decimal.NewFromInt(0),
			decimal.NewFromInt(7),
			decimal.NewFromInt(3),
		)
		require.True(t, got.IsZero(), "zero amount must convert to zero")
	})

	t.Run("negative amount preserves sign", func(t *testing.T) {
		got := convertWithRates(
			decimal.NewFromInt(-100),
			decimal.NewFromInt(1),
			decimal.NewFromInt(2),
		)
		require.Equal(t, "-200", got.String())
	})
}

func TestParseFawazahmed0Response(t *testing.T) {
	t.Parallel()

	t.Run("parses and uppercases quotes", func(t *testing.T) {
		body := []byte(`{"date":"2026-09-07","usd":{"cny":7.2,"jpy":150.0,"eur":0.92}}`)
		rates, err := parseFawazahmed0Response(body, "usd")
		require.NoError(t, err)
		require.True(t, rates["CNY"].Equal(decimal.RequireFromString("7.2")))
		require.True(t, rates["JPY"].Equal(decimal.NewFromInt(150)))
		require.True(t, rates["EUR"].Equal(decimal.RequireFromString("0.92")))
		require.True(t, rates["USD"].Equal(decimal.NewFromInt(1)), "anchor identity must be injected")
	})

	t.Run("missing base key", func(t *testing.T) {
		_, err := parseFawazahmed0Response([]byte(`{"date":"2026-09-07","eur":{}}`), "usd")
		require.Error(t, err)
	})

	t.Run("malformed json", func(t *testing.T) {
		_, err := parseFawazahmed0Response([]byte(`not json`), "usd")
		require.Error(t, err)
	})

	t.Run("non-numeric quote", func(t *testing.T) {
		_, err := parseFawazahmed0Response([]byte(`{"date":"2026-09-07","usd":{"cny":"abc"}}`), "usd")
		require.Error(t, err)
	})
}

func TestAnchorCurrencyDefault(t *testing.T) {
	// Environment override is intentionally not mutated here so the test stays
	// hermetic; the default is asserted.
	require.Equal(t, "USD", AnchorCurrency())
}
