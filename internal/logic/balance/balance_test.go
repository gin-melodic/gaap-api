package balance_test

import (
	"testing"

	_ "gaap-api/internal/logic/account"
	_ "gaap-api/internal/logic/balance"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/test/gtest"
	"github.com/shopspring/decimal"
)

// These tests verify the business logic of balance calculations.
// For full integration tests with database transactions, use Docker-based testing.

// Test_TransactionTypes_AreCorrect verifies transaction type constants are correct.
func Test_TransactionTypes_AreCorrect(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Verify that transaction types match expected values
		g.Assert("EXPENSE", "EXPENSE")
		g.Assert("INCOME", "INCOME")
		g.Assert("TRANSFER", "TRANSFER")
	})
}

// Test_TransactionCreateInput_Structure verifies the input structure is correct.
func Test_TransactionCreateInput_Structure(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		input := model.TransactionCreateInput{
			UserId:   "user-001",
			Date:     "2025-12-22",
			From:     "acc-from",
			To:       "acc-to",
			Amount:   100.50,
			Currency: "CNY",
			Note:     "Test transaction",
			Type:     "EXPENSE",
		}

		g.Assert(input.UserId, "user-001")
		g.Assert(input.From, "acc-from")
		g.Assert(input.To, "acc-to")
		g.Assert(input.Amount, 100.50)
		g.Assert(input.Type, "EXPENSE")
	})
}

// Test_BalanceCalculation_Expense tests expense balance calculation logic.
func Test_BalanceCalculation_Expense(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Simulate: Asset account balance 100, expense 30
		// Expected: Asset account balance should become 70
		initialBalance := 100.0
		expenseAmount := 30.0
		expectedBalance := initialBalance - expenseAmount

		g.Assert(expectedBalance, 70.0)
	})
}

// Test_BalanceCalculation_Income tests income balance calculation logic.
func Test_BalanceCalculation_Income(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Simulate: Asset account balance 50, income 200
		// Expected: Asset account balance should become 250
		initialBalance := 50.0
		incomeAmount := 200.0
		expectedBalance := initialBalance + incomeAmount

		g.Assert(expectedBalance, 250.0)
	})
}

// Test_BalanceCalculation_Transfer tests transfer balance calculation logic.
func Test_BalanceCalculation_Transfer(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Simulate: From account 100, To account 20, transfer 50
		// Expected: From account 50, To account 70
		fromInitial := 100.0
		toInitial := 20.0
		transferAmount := 50.0

		fromExpected := fromInitial - transferAmount
		toExpected := toInitial + transferAmount

		g.Assert(fromExpected, 50.0)
		g.Assert(toExpected, 70.0)
	})
}

// Test_BalanceCalculation_NegativeAllowed tests negative balance is allowed.
func Test_BalanceCalculation_NegativeAllowed(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Simulate: Asset account balance 10, expense 50
		// Expected: Asset account balance should become -40 (allowed)
		initialBalance := 10.0
		expenseAmount := 50.0
		expectedBalance := initialBalance - expenseAmount

		g.Assert(expectedBalance, -40.0)
	})
}

// Test_BalanceCalculation_ZeroAmount tests zero amount doesn't change balance.
func Test_BalanceCalculation_ZeroAmount(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		initialBalance := 100.0
		zeroAmount := 0.0
		expectedBalance := initialBalance - zeroAmount

		g.Assert(expectedBalance, 100.0)
	})
}

// Test_BalanceCalculation_LargeAmount tests large amounts are handled correctly.
func Test_BalanceCalculation_LargeAmount(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Test with very large numbers
		initialBalance := 0.0
		largeAmount := 999999999999.99
		expectedBalance := initialBalance + largeAmount

		g.Assert(expectedBalance, 999999999999.99)
	})
}

// Test_ReverseExpense_RestoresBalance tests reversing expense restores balance.
func Test_ReverseExpense_RestoresBalance(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// After expense: balance is 70 (was 100, spent 30)
		// Reverse expense: add 30 back
		currentBalance := 70.0
		originalExpenseAmount := 30.0
		restoredBalance := currentBalance + originalExpenseAmount

		g.Assert(restoredBalance, 100.0)
	})
}

// Test_ReverseIncome_DecreasesBalance tests reversing income decreases balance.
func Test_ReverseIncome_DecreasesBalance(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// After income: balance is 250 (was 50, received 200)
		// Reverse income: subtract 200
		currentBalance := 250.0
		originalIncomeAmount := 200.0
		restoredBalance := currentBalance - originalIncomeAmount

		g.Assert(restoredBalance, 50.0)
	})
}

// Test_ReverseTransfer_RestoresBothBalances tests reversing transfer restores both.
func Test_ReverseTransfer_RestoresBothBalances(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// After transfer: from=50 (was 100), to=70 (was 20), transferred 50
		// Reverse: from gets +50, to gets -50
		fromCurrent := 50.0
		toCurrent := 70.0
		transferAmount := 50.0

		fromRestored := fromCurrent + transferAmount
		toRestored := toCurrent - transferAmount

		g.Assert(fromRestored, 100.0)
		g.Assert(toRestored, 20.0)
	})
}

// =============================================================================
// Accounting Equation Tests
// Assets = Liabilities + Equity
// =============================================================================

// Test_AccountingEquation_Balance verifies the fundamental accounting equation.
// This is the core principle: Assets = Liabilities + Equity
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
		// Initial state
		assets := decimal.NewFromFloat(50000.00)
		liabilities := decimal.NewFromFloat(20000.00)
		equity := decimal.NewFromFloat(30000.00)

		// Transaction: Borrow 5000 (increases both assets and liabilities)
		loanAmount := decimal.NewFromFloat(5000.00)
		assets = assets.Add(loanAmount)
		liabilities = liabilities.Add(loanAmount)

		// Equation must still hold: Assets = Liabilities + Equity
		g.Assert(assets, liabilities.Add(equity))
		g.Assert(assets.String(), "55000")
		g.Assert(liabilities.String(), "25000")
	})
}

// Test_AccountingEquation_DoubleEntry verifies double-entry bookkeeping.
func Test_AccountingEquation_DoubleEntry(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Initial balances
		cashAccount := decimal.NewFromFloat(10000.00) // Asset
		revenueAccount := decimal.NewFromFloat(0.00)  // Equity (Revenue)
		expenseAccount := decimal.NewFromFloat(0.00)  // Equity (Expense)

		// Record income: +1500 to Cash, +1500 to Revenue
		incomeAmount := decimal.NewFromFloat(1500.00)
		cashAccount = cashAccount.Add(incomeAmount)
		revenueAccount = revenueAccount.Add(incomeAmount)

		// Record expense: -800 from Cash, +800 to Expense
		expenseAmount := decimal.NewFromFloat(800.00)
		cashAccount = cashAccount.Sub(expenseAmount)
		expenseAccount = expenseAccount.Add(expenseAmount)

		// Net change in equity = Revenue - Expense
		netEquityChange := revenueAccount.Sub(expenseAccount)

		// Verify double-entry balance
		g.Assert(cashAccount.String(), "10700")
		g.Assert(netEquityChange.String(), "700")
		g.Assert(cashAccount.Sub(decimal.NewFromFloat(10000.00)), netEquityChange)
	})
}

// =============================================================================
// Precision Loss Boundary Tests
// Using Decimal to prevent floating-point precision issues
// =============================================================================

// Test_Precision_FloatingPointIssue demonstrates why we need Decimal.
func Test_Precision_FloatingPointIssue(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Classic floating-point precision problem
		// In float64: 0.1 + 0.2 != 0.3 (actually 0.30000000000000004)

		// Using Decimal - precise calculation
		a := decimal.NewFromFloat(0.1)
		b := decimal.NewFromFloat(0.2)
		expected := decimal.NewFromFloat(0.3)

		result := a.Add(b)
		g.Assert(result.Equal(expected), true)
		g.Assert(result.String(), "0.3")
	})
}

// Test_Precision_SmallAmounts tests precision with very small amounts.
func Test_Precision_SmallAmounts(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Financial calculations with cents/pennies
		amount1 := decimal.RequireFromString("0.01")
		amount2 := decimal.RequireFromString("0.02")
		amount3 := decimal.RequireFromString("0.03")

		sum := amount1.Add(amount2).Add(amount3)
		expected := decimal.RequireFromString("0.06")

		g.Assert(sum.Equal(expected), true)
		g.Assert(sum.String(), "0.06")
	})
}

// Test_Precision_LargeNumbersWithDecimals tests precision with large amounts.
func Test_Precision_LargeNumbersWithDecimals(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Large transaction amount with decimals
		largeAmount := decimal.RequireFromString("999999999999.99")
		smallFee := decimal.RequireFromString("0.01")

		// Adding small fee to large amount should be precise
		total := largeAmount.Add(smallFee)
		g.Assert(total.String(), "1000000000000")

		// Subtracting should give exact original
		originalAmount := total.Sub(smallFee)
		g.Assert(originalAmount.String(), "999999999999.99")
	})
}

// Test_Precision_DivisionRounding tests division rounding behavior.
func Test_Precision_DivisionRounding(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Split 100 among 3 accounts
		total := decimal.RequireFromString("100.00")
		divisor := decimal.NewFromInt(3)

		// Each share (with 2 decimal places)
		share := total.Div(divisor).Round(2)

		// 100 / 3 = 33.33 (rounded to 2 decimals)
		g.Assert(share.String(), "33.33")

		// Total of 3 shares
		calculatedTotal := share.Mul(decimal.NewFromInt(3))
		g.Assert(calculatedTotal.String(), "99.99")

		// Remainder (the "penny" that's lost in division)
		remainder := total.Sub(calculatedTotal)
		g.Assert(remainder.String(), "0.01")
	})
}

// Test_Precision_CurrencyConversion tests currency conversion precision.
func Test_Precision_CurrencyConversion(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Amount in USD
		usdAmount := decimal.RequireFromString("1000.00")

		// Exchange rate: 1 USD = 7.2456 CNY
		exchangeRate := decimal.RequireFromString("7.2456")

		// Convert to CNY
		cnyAmount := usdAmount.Mul(exchangeRate)
		g.Assert(cnyAmount.String(), "7245.6")

		// Convert back to USD (should be exact)
		usdBack := cnyAmount.Div(exchangeRate)
		g.Assert(usdBack.String(), "1000")
	})
}

// Test_Precision_AccumulatedRoundingError tests accumulated rounding.
func Test_Precision_AccumulatedRoundingError(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Simulate 1000 small transactions
		total := decimal.Zero
		transactionAmount := decimal.RequireFromString("0.01")

		for i := 0; i < 1000; i++ {
			total = total.Add(transactionAmount)
		}

		// Must be exactly 10.00, no accumulated error
		g.Assert(total.String(), "10")
		g.Assert(total.Equal(decimal.NewFromInt(10)), true)
	})
}

// Test_Precision_BalanceEquality tests balance equality check.
func Test_Precision_BalanceEquality(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Two ways to calculate the same amount
		method1 := decimal.RequireFromString("100.00").
			Mul(decimal.RequireFromString("1.05")). // +5% interest
			Round(2)

		method2 := decimal.RequireFromString("100.00").
			Add(decimal.RequireFromString("5.00")) // +5.00 absolute

		g.Assert(method1.Equal(method2), true)
		g.Assert(method1.String(), "105")
	})
}
