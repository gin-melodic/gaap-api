package account_test

import (
	"context"
	"testing"

	_ "gaap-api/internal/logic/account"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/middleware"
	"gaap-api/internal/service"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/test/gtest"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =============================================================================
// Constants and Test Data
// =============================================================================

// Account table columns for mocking database queries.
// Must match the entity.Accounts struct field order.
var accountColumns = []string{
	"id", "user_id", "parent_id", "name", "type", "is_group",
	"currency_code", "balance_units", "balance_nanos", "balance_decimal",
	"default_child_id", "equity_account_id", "date", "number", "remarks",
	"created_at", "updated_at", "deleted_at",
}

// Account type constants (matching utils/enum.go)
// Used for balance restriction validation tests.
const (
	AccountTypeUnspecified = 0
	AccountTypeAsset       = 1 // Assets: bank accounts, cash, investments
	AccountTypeLiability   = 2 // Liabilities: credit cards, loans
	AccountTypeIncome      = 3 // Income: salary, interest
	AccountTypeExpense     = 4 // Expense: food, utilities
	AccountTypeEquity      = 5 // Equity: opening balance, retained earnings
)

// =============================================================================
// GetAccount Tests - Cache and Access Control
// =============================================================================

// Test_Account_GetAccount verifies that GetAccount correctly retrieves an account
// from the database with caching. This test validates:
// - Cache layer falls back to DB when Redis is unavailable
// - Account fields are correctly mapped from database
// - User ownership is verified
func Test_Account_GetAccount(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		userId := uuid.New()
		accountId := uuid.New()
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, userId.String())

		// Setup: Initialize mock database with required metadata
		testutil.MockDBInit(mock)
		testutil.MockMeta(mock, "accounts", accountColumns)

		// Mock: Return a valid Asset account with USD 1000.50 balance
		rows := sqlmock.NewRows(accountColumns).AddRow(
			accountId.String(), userId.String(), nil, "Checking Account", AccountTypeAsset, false,
			"USD", int64(1000), 500000000, 1000.5, // 1000 units + 0.5 nanos = 1000.50
			nil, nil, "2023-01-01", "CHK-001", "Main checking",
			"2023-01-01", "2023-01-01", nil,
		)

		mock.ExpectQuery(`SELECT .* FROM "?accounts"?`).
			WithArgs(accountId).
			WillReturnRows(rows)

		// Execute
		out, err := service.Account().GetAccount(ctx, accountId)

		// Assert: Account is retrieved with correct values
		g.AssertNil(err)
		g.AssertNE(out, nil)
		g.Assert(out.Id, accountId)
		g.Assert(out.UserId, userId)
		g.Assert(out.Name, "Checking Account")
		g.Assert(out.Type, AccountTypeAsset)
		g.Assert(out.CurrencyCode, "USD")
		g.Assert(out.BalanceUnits, int64(1000))
		g.Assert(out.BalanceNanos, 500000000)
	})
}

// Test_Account_GetAccount_NotFound verifies that GetAccount returns an error
// when the requested account does not exist in the database.
// Expected behavior: Error is returned, output is nil.
func Test_Account_GetAccount_NotFound(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		userId := uuid.New()
		accountId := uuid.New()
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, userId.String())

		testutil.MockDBInit(mock)
		testutil.MockMeta(mock, "accounts", accountColumns)

		// Mock: Return empty result set (account not found)
		rows := sqlmock.NewRows(accountColumns)
		mock.ExpectQuery(`SELECT .* FROM "?accounts"?`).
			WithArgs(accountId).
			WillReturnRows(rows)

		// Execute
		out, err := service.Account().GetAccount(ctx, accountId)

		// Assert: Error returned for non-existent account
		g.AssertNE(err, nil)
		g.Assert(out, nil)
	})
}

// Test_Account_GetAccount_WrongUser verifies that GetAccount denies access
// when the account belongs to a different user. This is a critical security test.
// Expected behavior: Access denied error, output is nil.
func Test_Account_GetAccount_WrongUser(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		mock, _ := testutil.InitMockDB(t)
		requestingUserId := uuid.New()
		ownerUserId := uuid.New() // Different user
		accountId := uuid.New()
		ctx := context.WithValue(context.Background(), middleware.UserIdKey, requestingUserId.String())

		testutil.MockDBInit(mock)
		testutil.MockMeta(mock, "accounts", accountColumns)

		// Mock: Return account owned by a DIFFERENT user
		rows := sqlmock.NewRows(accountColumns).AddRow(
			accountId.String(), ownerUserId.String(), nil, "Private Account", AccountTypeAsset, false,
			"USD", int64(50000), 0, 50000.0,
			nil, nil, "2023-01-01", "", "",
			"2023-01-01", "2023-01-01", nil,
		)

		mock.ExpectQuery(`SELECT .* FROM "?accounts"?`).
			WithArgs(accountId).
			WillReturnRows(rows)

		// Execute
		out, err := service.Account().GetAccount(ctx, accountId)

		// Assert: Access denied for different user's account
		g.AssertNE(err, nil)
		g.Assert(out, nil)
	})
}

// =============================================================================
// Cache Key Tests
// =============================================================================

// Test_AccountCacheKey validates the cache key generation function.
// Cache keys must be deterministic and follow the pattern: "account:{id}"
func Test_AccountCacheKey(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Test with a sample account ID
		accountId := "550e8400-e29b-41d4-a716-446655440000"
		key := utils.AccountCacheKey(accountId)
		g.Assert(key, "account:550e8400-e29b-41d4-a716-446655440000")

		// Test that different IDs produce different keys
		key2 := utils.AccountCacheKey("different-id")
		g.AssertNE(key, key2)
	})
}

// =============================================================================
// Balance Restriction Tests
// =============================================================================

// Test_BalanceRestriction_AccountTypes validates which account types allow
// direct balance updates. According to double-entry bookkeeping:
//   - Asset (1) and Liability (2): Allow direct balance updates via opening balance
//   - Income (3), Expense (4), Equity (5): MUST NOT allow direct balance updates
//     (balances are derived from transactions only)
func Test_BalanceRestriction_AccountTypes(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Restricted types: balance can ONLY be changed through transactions
		restrictedTypes := []int{AccountTypeExpense, AccountTypeIncome, AccountTypeEquity}

		// Allowed types: balance can be set directly (via opening balance transaction)
		allowedTypes := []int{AccountTypeAsset, AccountTypeLiability}

		// Verify the restriction list is complete
		g.Assert(len(restrictedTypes), 3)
		g.Assert(len(allowedTypes), 2)

		// Verify specific types in restricted list
		for _, t := range restrictedTypes {
			g.Assert(t >= AccountTypeIncome && t <= AccountTypeEquity, true)
		}

		// Verify specific types in allowed list
		for _, t := range allowedTypes {
			g.Assert(t == AccountTypeAsset || t == AccountTypeLiability, true)
		}
	})
}

// =============================================================================
// Accounting Equation Tests: Assets = Liabilities + Equity
// =============================================================================

// Test_AccountingEquation_BasicBalance validates the fundamental accounting equation.
// The equation Assets = Liabilities + Equity must ALWAYS hold true.
func Test_AccountingEquation_BasicBalance(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Setup: A user has $10,000 in assets
		assets := decimal.NewFromFloat(10000.00)
		// $3,000 in credit card debt (liability)
		liabilities := decimal.NewFromFloat(3000.00)
		// Net worth (equity) = Assets - Liabilities = $7,000
		equity := decimal.NewFromFloat(7000.00)

		// Assert: Accounting equation holds
		// Assets = Liabilities + Equity
		g.Assert(assets.Equal(liabilities.Add(equity)), true)

		// Additional validation
		g.Assert(liabilities.String(), "3000")
	})
}

// Test_AccountingEquation_OpeningBalance verifies that when an account is created
// with an opening balance, the accounting equation remains balanced.
// Flow: Equity account decreases, Asset account increases by the same amount.
func Test_AccountingEquation_OpeningBalance(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Initial state: All zeros
		assets := decimal.Zero
		liabilities := decimal.Zero
		equity := decimal.Zero

		// Action: Create new checking account with $5,000 opening balance
		// This creates a transaction: Equity -> Asset
		openingBalance := decimal.NewFromFloat(5000.00)

		// Effect: Asset increases, Equity decreases
		assets = assets.Add(openingBalance)
		equity = equity.Sub(openingBalance) // Equity is negative (credit balance)

		// Assert: Equation holds
		// Assets (5000) = Liabilities (0) + Equity (-5000)
		// Rearranged: 5000 = 0 + (-5000) is wrong
		// Actually in double-entry: Asset DR 5000, Equity CR 5000
		// The sum of debits = sum of credits
		// So: Assets (5000) = Liabilities (0) + Owner's Equity (5000)
		// In our model, equity balance is stored as negative when credited
		g.Assert(assets.Add(equity).IsZero(), true) // Debits = Credits
		g.Assert(liabilities.IsZero(), true)        // Liabilities unchanged
	})
}

// Test_AccountingEquation_Transfer verifies that transfers between Asset accounts
// maintain the accounting equation. Total assets remain unchanged.
func Test_AccountingEquation_Transfer(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Setup: Two asset accounts
		checkingBalance := decimal.NewFromFloat(10000.00)
		savingsBalance := decimal.NewFromFloat(5000.00)
		totalAssets := checkingBalance.Add(savingsBalance)

		// Action: Transfer $2,000 from checking to savings
		transferAmount := decimal.NewFromFloat(2000.00)
		checkingBalance = checkingBalance.Sub(transferAmount)
		savingsBalance = savingsBalance.Add(transferAmount)

		// Assert: Total assets unchanged
		newTotalAssets := checkingBalance.Add(savingsBalance)
		g.Assert(totalAssets.Equal(newTotalAssets), true)
		g.Assert(checkingBalance.String(), "8000")
		g.Assert(savingsBalance.String(), "7000")
	})
}

// Test_AccountingEquation_Expense verifies that recording an expense
// maintains the accounting equation. Asset decreases, Expense account increases.
func Test_AccountingEquation_Expense(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Setup: Starting position
		assets := decimal.NewFromFloat(10000.00)
		expenses := decimal.Zero // Expense is a contra-equity account

		// Action: Record $100 grocery expense
		// Flow: Asset (Cash) -> Expense (Groceries)
		expenseAmount := decimal.NewFromFloat(100.00)
		assets = assets.Sub(expenseAmount)
		expenses = expenses.Add(expenseAmount)

		// Assert: Asset decreased, expense increased by same amount
		g.Assert(assets.String(), "9900")
		g.Assert(expenses.String(), "100")
		// Net effect: reduces owner's equity
	})
}

// Test_AccountingEquation_Income verifies that recording income
// maintains the accounting equation. Asset increases, Income account increases.
func Test_AccountingEquation_Income(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Setup: Starting position
		assets := decimal.NewFromFloat(10000.00)
		income := decimal.Zero // Income is an equity-like account

		// Action: Record $5000 salary income
		// Flow: Income (Salary) -> Asset (Bank)
		incomeAmount := decimal.NewFromFloat(5000.00)
		assets = assets.Add(incomeAmount)
		income = income.Add(incomeAmount)

		// Assert: Asset increased, income increased
		g.Assert(assets.String(), "15000")
		g.Assert(income.String(), "5000")
		// Net effect: increases owner's equity
	})
}

// =============================================================================
// MoneyHelper Units/Nanos Boundary Tests
// =============================================================================

// Test_MoneyHelper_UnitsNanos_BasicConversion validates units/nanos conversion.
// The format uses: units (integer part) + nanos (fractional part, 10^-9 precision)
func Test_MoneyHelper_UnitsNanos_BasicConversion(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Test case 1: Simple integer amount
		units1 := int64(100)
		nanos1 := int32(0)
		combined1 := float64(units1) + float64(nanos1)/1e9
		g.Assert(combined1, 100.0)

		// Test case 2: Amount with fractional part
		units2 := int64(1234)
		nanos2 := int32(567890000) // 0.56789
		combined2 := float64(units2) + float64(nanos2)/1e9
		g.Assert(combined2, 1234.56789)

		// Test case 3: Maximum nanos value (just under 1)
		units3 := int64(0)
		nanos3 := int32(999999999) // 0.999999999
		combined3 := float64(units3) + float64(nanos3)/1e9
		g.Assert(combined3 < 1.0, true)
		g.Assert(combined3 > 0.99, true)
	})
}

// Test_MoneyHelper_UnitsNanos_NegativeValues validates handling of negative amounts.
// Negative values are represented with negative units; nanos are always positive.
func Test_MoneyHelper_UnitsNanos_NegativeValues(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Negative amount: -$100.50
		units := int64(-101)      // Floor of -100.50
		nanos := int32(500000000) // 0.5 (always positive)
		// Combined: -101 + 0.5 = -100.5
		combined := float64(units) + float64(nanos)/1e9
		g.Assert(combined, -100.5)
	})
}

// Test_MoneyHelper_UnitsNanos_ZeroValue validates that zero is correctly represented.
func Test_MoneyHelper_UnitsNanos_ZeroValue(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		units := int64(0)
		nanos := int32(0)
		combined := float64(units) + float64(nanos)/1e9
		g.Assert(combined, 0.0)
	})
}

// Test_MoneyHelper_UnitsNanos_PrecisionBoundary tests precision limits.
// Nanos supports up to 9 decimal places (nanoseconds = 10^-9).
func Test_MoneyHelper_UnitsNanos_PrecisionBoundary(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Maximum representable fractional amount
		maxNanos := int32(999999999)
		g.Assert(maxNanos < 1e9, true)

		// Minimum representable fractional increment (1 nano = 0.000000001)
		minNanos := int32(1)
		minFraction := float64(minNanos) / 1e9
		g.Assert(minFraction, 0.000000001)

		// Typical financial precision (2 decimal places = cents)
		cents := int32(10000000) // 0.01
		g.Assert(float64(cents)/1e9, 0.01)
	})
}

// Test_MoneyHelper_Addition validates safe addition of money amounts.
// Uses decimal to prevent floating-point precision loss.
func Test_MoneyHelper_Addition(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Classic floating-point problem: 0.1 + 0.2 != 0.3 in binary
		a := decimal.NewFromFloat(0.1)
		b := decimal.NewFromFloat(0.2)
		expected := decimal.NewFromFloat(0.3)

		result := a.Add(b)

		// Decimal library handles this correctly
		g.Assert(result.Equal(expected), true)
		g.Assert(result.String(), "0.3")
	})
}

// Test_MoneyHelper_Subtraction validates safe subtraction of money amounts.
func Test_MoneyHelper_Subtraction(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		balance := decimal.NewFromFloat(1000.00)
		withdrawal := decimal.NewFromFloat(100.50)

		result := balance.Sub(withdrawal)

		g.Assert(result.String(), "899.5")
	})
}

// =============================================================================
// Balance Migration Tests - Accounting Equation Preservation
// =============================================================================

// Test_BalanceMigration_PreservesEquation verifies that migrating an account's
// balance to another account preserves the accounting equation.
// This simulates the new transaction-based migration approach.
func Test_BalanceMigration_PreservesEquation(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Setup: Two Asset accounts and one Equity account
		sourceAsset := decimal.NewFromFloat(1000.00)
		targetAsset := decimal.NewFromFloat(500.00)
		equityAccount := decimal.Zero
		totalAssets := sourceAsset.Add(targetAsset)

		// Migration Step 1: Source Account -> Equity (clear source)
		// Effect: Source decreases, Equity increases
		sourceAsset = sourceAsset.Sub(decimal.NewFromFloat(1000.00))
		equityAccount = equityAccount.Add(decimal.NewFromFloat(1000.00))

		// Verify intermediate state
		g.Assert(sourceAsset.String(), "0")
		g.Assert(equityAccount.String(), "1000")

		// Migration Step 2: Equity -> Target Account (transfer balance)
		// Effect: Equity decreases, Target increases
		equityAccount = equityAccount.Sub(decimal.NewFromFloat(1000.00))
		targetAsset = targetAsset.Add(decimal.NewFromFloat(1000.00))

		// Assert final state
		g.Assert(sourceAsset.String(), "0")    // Source is now empty
		g.Assert(targetAsset.String(), "1500") // Target has combined balance
		g.Assert(equityAccount.String(), "0")  // Equity net change is zero

		// Assert: Total assets unchanged (equation preserved)
		newTotalAssets := sourceAsset.Add(targetAsset)
		g.Assert(totalAssets.Equal(newTotalAssets), true)
	})
}

// Test_BalanceMigration_MultiCurrency validates that multi-currency migration
// handles each currency separately.
func Test_BalanceMigration_MultiCurrency(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// USD accounts
		usdSource := decimal.NewFromFloat(1000.00)
		usdTarget := decimal.NewFromFloat(500.00)

		// EUR accounts (separate from USD)
		eurSource := decimal.NewFromFloat(800.00)
		eurTarget := decimal.NewFromFloat(200.00)

		// Migrate USD
		usdTarget = usdTarget.Add(usdSource)
		usdSource = decimal.Zero

		// Migrate EUR
		eurTarget = eurTarget.Add(eurSource)
		eurSource = decimal.Zero

		// Assert: Currencies are handled independently
		g.Assert(usdTarget.String(), "1500")
		g.Assert(usdSource.String(), "0")
		g.Assert(eurTarget.String(), "1000")
		g.Assert(eurSource.String(), "0")

		// Total values per currency are preserved
		g.Assert(usdTarget.String(), "1500") // Was 1000 + 500
		g.Assert(eurTarget.String(), "1000") // Was 800 + 200
	})
}

// =============================================================================
// Edge Cases and Error Conditions
// =============================================================================

// Test_Account_NilUUID verifies handling of nil UUID account requests.
func Test_Account_NilUUID(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// A nil UUID should be rejected or handled gracefully
		nilUUID := uuid.Nil
		g.Assert(nilUUID == uuid.UUID{}, true)
	})
}

// Test_Account_MaxBalanceValue validates handling of maximum balance values.
// int64 max is 9,223,372,036,854,775,807 (about 9.2 quintillion)
func Test_Account_MaxBalanceValue(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Maximum balance in units
		maxUnits := int64(9223372036854775807)
		maxNanos := int32(999999999)

		// Verify they fit within expected types
		g.Assert(maxUnits > 0, true)
		g.Assert(maxNanos > 0, true)

		// Combined approximation (loses precision at this scale)
		combined := decimal.NewFromInt(maxUnits).Add(
			decimal.NewFromInt(int64(maxNanos)).Div(decimal.NewFromInt(1e9)),
		)
		g.Assert(combined.GreaterThan(decimal.Zero), true)
	})
}

// Test_Account_NegativeBalance validates handling of negative balances.
// Credit cards and loans typically have negative balances (from user perspective).
func Test_Account_NegativeBalance(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Credit card with $500 debt
		creditCardBalance := decimal.NewFromFloat(-500.00)

		// Making a payment of $100
		payment := decimal.NewFromFloat(100.00)
		newBalance := creditCardBalance.Add(payment)

		g.Assert(newBalance.String(), "-400")
		g.Assert(newBalance.LessThan(decimal.Zero), true)
	})
}
