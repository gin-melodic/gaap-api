package balance_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestExpandedMoneyPrecisionBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		units int64
		nanos int32
		want  string
	}{
		{name: "large UAT transaction", units: 999_999_999_999, nanos: 990_000_000, want: "999999999999.99"},
		{name: "large UAT opening balance", units: 999_999_999_999_999_999, nanos: 990_000_000, want: "999999999999999999.99"},
		{name: "maximum positive units", units: 9_223_372_036_854_775_807, nanos: 999_999_999, want: "9223372036854775807.999999999"},
		{name: "minimum negative units", units: -9_223_372_036_854_775_808, nanos: 0, want: "-9223372036854775808"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := decimal.NewFromInt(tt.units).Add(decimal.NewFromInt32(tt.nanos).Shift(-9))
			require.Equal(t, tt.want, value.String())
			require.LessOrEqual(t, len(value.Truncate(0).Abs().String()), 19)
		})
	}
}
