package transaction_test

import (
	"testing"

	_ "gaap-api/internal/logic/account"
	_ "gaap-api/internal/logic/balance"
	_ "gaap-api/internal/logic/transaction"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/test/gtest"
)

// Test_TransactionModel tests the transaction model structures.
func Test_TransactionModel(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Test TransactionCreateInput
		createInput := model.TransactionCreateInput{
			UserId:   "user-001",
			Date:     "2025-12-22",
			From:     "acc-from",
			To:       "acc-to",
			Amount:   100.50,
			Currency: "CNY",
			Note:     "Test transaction",
			Type:     "EXPENSE",
		}

		g.Assert(createInput.UserId, "user-001")
		g.Assert(createInput.From, "acc-from")
		g.Assert(createInput.To, "acc-to")
		g.Assert(createInput.Amount, 100.50)
		g.Assert(createInput.Type, "EXPENSE")
	})
}

// Test_TransactionUpdateInput tests the update input structure.
func Test_TransactionUpdateInput(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		updateInput := model.TransactionUpdateInput{
			Date:     "2025-12-23",
			From:     "acc-from-updated",
			To:       "acc-to-updated",
			Amount:   200.0,
			Currency: "USD",
			Note:     "Updated note",
			Type:     "INCOME",
		}

		g.Assert(updateInput.Amount, 200.0)
		g.Assert(updateInput.Type, "INCOME")
	})
}

// Test_TransactionQueryInput tests the query input structure.
func Test_TransactionQueryInput(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		queryInput := model.TransactionQueryInput{
			Page:      1,
			Limit:     20,
			StartDate: "2025-01-01",
			EndDate:   "2025-12-31",
			AccountId: "acc-001",
			Type:      "EXPENSE",
			SortBy:    "date",
			SortOrder: "desc",
		}

		g.Assert(queryInput.Page, 1)
		g.Assert(queryInput.Limit, 20)
		g.Assert(queryInput.Type, "EXPENSE")
		g.Assert(queryInput.SortOrder, "desc")
	})
}

// Test_TransactionOutput tests the transaction output structure.
func Test_TransactionOutput(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		tx := model.Transaction{
			Id:       "tx-001",
			Date:     "2025-12-22",
			From:     "acc-from",
			To:       "acc-to",
			Amount:   150.75,
			Currency: "CNY",
			Note:     "Test",
			Type:     "TRANSFER",
		}

		g.Assert(tx.Id, "tx-001")
		g.Assert(tx.From, "acc-from")
		g.Assert(tx.To, "acc-to")
		g.Assert(tx.Amount, 150.75)
		g.Assert(tx.Type, "TRANSFER")
	})
}

// Note: Integration tests for Create, Update, Delete operations
// should be run with Docker-based testing against a real database.
// These operations now use database transactions for balance synchronization.
//
// Integration test scenarios to cover:
// 1. Create EXPENSE -> verify from_account balance decreases
// 2. Create INCOME -> verify to_account balance increases
// 3. Create TRANSFER -> verify both account balances update
// 4. Update transaction amount -> verify balance adjustment
// 5. Delete transaction -> verify balance reversal
// 6. Transaction atomicity -> verify rollback on failure
