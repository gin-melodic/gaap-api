package utils

import (
	"errors"
	"gaap-api/internal/model/entity"

	"github.com/shopspring/decimal"
)

const (
	NanosMod = 1_000_000_000 // 10^9
)

// MoneyHelper use for packege calculate
type MoneyHelper struct {
	decimal.Decimal
	Currency string
}

// NewFromEntity create MoneyHelper from entity.Account
func NewFromEntity(e *entity.Accounts) *MoneyHelper {
	// units + (nanos * 10^-9)
	d := decimal.NewFromInt(e.BalanceUnits).Add(
		decimal.New(int64(e.BalanceNanos), -9),
	)
	return &MoneyHelper{
		Decimal:  d,
		Currency: e.CurrencyCode,
	}
}

// NewMoneyFromUnitsAndNanos create MoneyHelper from units and nanos
func NewMoneyFromUnitsAndNanos(units int64, nanos int32, currency string) *MoneyHelper {
	return &MoneyHelper{
		Decimal:  decimal.NewFromInt(units).Add(decimal.New(int64(nanos), -9)),
		Currency: currency,
	}
}

// ToEntityValues convert MoneyHelper to entity.Account values
func (m *MoneyHelper) ToEntityValues() (int64, int32) {
	// get units and nanos
	units := m.Decimal.IntPart()
	nanosDecimal := m.Decimal.Sub(decimal.NewFromInt(units)).Mul(decimal.NewFromInt(NanosMod))
	nanos := int32(nanosDecimal.IntPart())
	return units, nanos
}

// NewFromTransactions create MoneyHelper from entity.Transactions
func NewFromTransactions(t *entity.Transactions) *MoneyHelper {
	// units + (nanos * 10^-9)
	d := decimal.NewFromInt(t.BalanceUnits).Add(
		decimal.New(int64(t.BalanceNanos), -9),
	)
	return &MoneyHelper{
		Decimal:  d,
		Currency: t.CurrencyCode,
	}
}

// ToTransactionsValues convert MoneyHelper to entity.Transactions values
func (m *MoneyHelper) ToTransactionsValues() (int64, int32) {
	// get units and nanos
	units := m.Decimal.IntPart()
	nanosDecimal := m.Decimal.Sub(decimal.NewFromInt(units)).Mul(decimal.NewFromInt(NanosMod))
	nanos := int32(nanosDecimal.IntPart())
	return units, nanos
}

// ---------------------------------------------------------
// The arithmetic methods (support chain calls, or return a new object)
// ---------------------------------------------------------
// Add
func (m *MoneyHelper) Add(other *MoneyHelper) (*MoneyHelper, error) {
	if m.Currency != other.Currency {
		return nil, errors.New("currency mismatch: cannot add different currencies")
	}
	// use decimal high precision addition
	newDec := m.Decimal.Add(other.Decimal)
	return &MoneyHelper{Decimal: newDec, Currency: m.Currency}, nil
}

// Sub
func (m *MoneyHelper) Sub(other *MoneyHelper) (*MoneyHelper, error) {
	if m.Currency != other.Currency {
		return nil, errors.New("currency mismatch")
	}
	newDec := m.Decimal.Sub(other.Decimal)
	return &MoneyHelper{Decimal: newDec, Currency: m.Currency}, nil
}

// Mul (Use case: calculate interest)
// multiplier is a float64 usually
func (m *MoneyHelper) Mul(multiplier float64) *MoneyHelper {
	mulDec := decimal.NewFromFloat(multiplier)
	newDec := m.Decimal.Mul(mulDec)

	return &MoneyHelper{Decimal: newDec, Currency: m.Currency}
}

// Div (Use case: calculate interest)
// Note: Division must specify the precision, here we default to 9 decimal places (matching Nanos)
// Caution: It is dangerous to use division in financial systems (for example, 100 yuan / 3 people). In actual accounting scenarios, after division, there will often be "remainders". This requires a specialized **“Allocation Algorithm”**, not a simple Div.
func (m *MoneyHelper) Div(divisor float64) *MoneyHelper {
	divDec := decimal.NewFromFloat(divisor)

	// DivRound(d, precision)
	newDec := m.Decimal.DivRound(divDec, 9)

	return &MoneyHelper{Decimal: newDec, Currency: m.Currency}
}

// Equals
func (m *MoneyHelper) Equals(other *MoneyHelper) bool {
	if m.Currency != other.Currency {
		return false
	}
	return m.Decimal.Equal(other.Decimal)
}

// GreaterThan
func (m *MoneyHelper) GreaterThan(other *MoneyHelper) bool {
	if m.Currency != other.Currency {
		return false
	}
	return m.Decimal.GreaterThan(other.Decimal)
}

// LessThan
func (m *MoneyHelper) LessThan(other *MoneyHelper) bool {
	if m.Currency != other.Currency {
		return false
	}
	return m.Decimal.LessThan(other.Decimal)
}

// IsZero returns true if the money amount is zero
func (m *MoneyHelper) IsZero() bool {
	return m.Decimal.IsZero()
}
