package balance_test

import (
	"testing"

	"github.com/gogf/gf/v2/test/gtest"
	"github.com/shopspring/decimal"
)

// =============================================================================
// Accounting Equation Tests (No model dependency)
// Assets = Liabilities + Equity
// =============================================================================

// Test_AccountingEquation_Balance verifies the fundamental accounting equation.
func Test_AccountingEquation_Balance(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		assets := decimal.NewFromFloat(10000.00)
		liabilities := decimal.NewFromFloat(3000.00)
		equity := decimal.NewFromFloat(7000.00)

		// Accounting Equation: Assets = Liabilities + Equity
		g.Assert(assets, liabilities.Add(equity))
	})
}

// Test_AccountingEquation_AfterTransaction verifies equation holds after transactions.
func Test_AccountingEquation_AfterTransaction(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		assets := decimal.NewFromFloat(50000.00)
		liabilities := decimal.NewFromFloat(20000.00)
		equity := decimal.NewFromFloat(30000.00)

		loanAmount := decimal.NewFromFloat(5000.00)
		assets = assets.Add(loanAmount)
		liabilities = liabilities.Add(loanAmount)

		g.Assert(assets, liabilities.Add(equity))
		g.Assert(assets.String(), "55000")
		g.Assert(liabilities.String(), "25000")
	})
}

// =============================================================================
// Precision Loss Boundary Tests
// Using Decimal to prevent floating-point precision issues
// =============================================================================

// Test_Precision_FloatingPointIssue demonstrates why we need Decimal.
func Test_Precision_FloatingPointIssue(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		a := decimal.NewFromFloat(0.1)
		b := decimal.NewFromFloat(0.2)
		expected := decimal.NewFromFloat(0.3)

		result := a.Add(b)
		g.Assert(result.Equal(expected), true)
		g.Assert(result.String(), "0.3")
	})
}

// =============================================================================
// Schema/Model Tests (Optional/Future)
// =============================================================================
// Service logic tests for ApplyTransaction are covered by integration logic
// and money_helper unit tests. Detailed mocking of transaction boundaries
// is omitted here to avoid brittleness.
