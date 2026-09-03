package reconciliation

import (
	"context"
	"fmt"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountDifference describes a persisted account balance that does not match
// the balance reconstructed from active transactions.
type AccountDifference struct {
	AccountID  uuid.UUID `json:"accountId"`
	UserID     uuid.UUID `json:"userId"`
	Name       string    `json:"name"`
	Type       int       `json:"type"`
	Currency   string    `json:"currency"`
	Actual     string    `json:"actual"`
	Expected   string    `json:"expected"`
	Difference string    `json:"difference"`
}

// Report is the result of a read-only reconciliation run.
type Report struct {
	Passed              bool                `json:"passed"`
	AccountsChecked     int                 `json:"accountsChecked"`
	TransactionsChecked int                 `json:"transactionsChecked"`
	Differences         []AccountDifference `json:"differences"`
	Issues              []string            `json:"issues"`
}

// Run reads a consistent database snapshot and compares every active account
// balance with the balance reconstructed from active transactions. The
// transaction is explicitly read-only so this command can never repair data.
func Run(ctx context.Context) (*Report, error) {
	var report *Report
	err := g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY"); err != nil {
			return gerror.Wrap(err, "failed to enforce read-only reconciliation")
		}

		var accounts []entity.Accounts
		accountColumns := dao.Accounts.Columns()
		if err := tx.Model(dao.Accounts.Table()).
			Fields(
				accountColumns.Id,
				accountColumns.UserId,
				accountColumns.Name,
				accountColumns.Type,
				accountColumns.CurrencyCode,
				accountColumns.BalanceUnits,
				accountColumns.BalanceNanos,
			).
			WhereNull(accountColumns.DeletedAt).
			Scan(&accounts); err != nil {
			return gerror.Wrap(err, "failed to read accounts")
		}

		var transactions []entity.Transactions
		transactionColumns := dao.Transactions.Columns()
		if err := tx.Model(dao.Transactions.Table()).
			Fields(
				transactionColumns.Id,
				transactionColumns.UserId,
				transactionColumns.FromAccountId,
				transactionColumns.ToAccountId,
				transactionColumns.CurrencyCode,
				transactionColumns.BalanceUnits,
				transactionColumns.BalanceNanos,
				transactionColumns.Type,
			).
			WhereNull(transactionColumns.DeletedAt).
			Scan(&transactions); err != nil {
			return gerror.Wrap(err, "failed to read transactions")
		}

		report = Reconcile(accounts, transactions)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

// Reconcile reconstructs account balances without modifying its inputs.
func Reconcile(accounts []entity.Accounts, transactions []entity.Transactions) *Report {
	report := &Report{
		AccountsChecked:     len(accounts),
		TransactionsChecked: len(transactions),
		Differences:         make([]AccountDifference, 0),
		Issues:              make([]string, 0),
	}

	accountByID := make(map[uuid.UUID]entity.Accounts, len(accounts))
	expected := make(map[uuid.UUID]decimal.Decimal, len(accounts))
	for _, account := range accounts {
		accountByID[account.Id] = account
		expected[account.Id] = decimal.Zero
	}

	for _, transaction := range transactions {
		if !validTransactionType(transaction.Type) {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s has unsupported type %d", transaction.Id, transaction.Type))
			continue
		}

		if transaction.BalanceNanos < -999_999_999 || transaction.BalanceNanos > 999_999_999 ||
			(transaction.BalanceUnits > 0 && transaction.BalanceNanos < 0) ||
			(transaction.BalanceUnits < 0 && transaction.BalanceNanos > 0) {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s has a non-canonical amount", transaction.Id))
			continue
		}
		amount := decimal.NewFromInt(transaction.BalanceUnits).Add(decimal.New(int64(transaction.BalanceNanos), -9))
		if amount.IsZero() {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s has a zero amount", transaction.Id))
			continue
		}
		if amount.IsNegative() && transaction.Type != utils.TransactionTypeOpeningBalance {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s has a negative amount", transaction.Id))
			continue
		}

		fromAccount, fromOK := accountByID[transaction.FromAccountId]
		toAccount, toOK := accountByID[transaction.ToAccountId]
		if !fromOK {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s references missing from account %s", transaction.Id, transaction.FromAccountId))
		}
		if !toOK {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s references missing to account %s", transaction.Id, transaction.ToAccountId))
		}
		if !fromOK || !toOK {
			continue
		}
		if fromAccount.UserId != transaction.UserId || toAccount.UserId != transaction.UserId {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s crosses user ownership", transaction.Id))
			continue
		}
		if fromAccount.CurrencyCode != transaction.CurrencyCode || toAccount.CurrencyCode != transaction.CurrencyCode {
			report.Issues = append(report.Issues, fmt.Sprintf("transaction %s has an account currency mismatch", transaction.Id))
			continue
		}

		expected[fromAccount.Id] = expected[fromAccount.Id].Sub(amount)
		expected[toAccount.Id] = expected[toAccount.Id].Add(amount)
	}

	for _, account := range accounts {
		actual := decimal.NewFromInt(account.BalanceUnits).Add(decimal.New(int64(account.BalanceNanos), -9))
		expectedBalance := expected[account.Id]
		if actual.Equal(expectedBalance) {
			continue
		}
		report.Differences = append(report.Differences, AccountDifference{
			AccountID:  account.Id,
			UserID:     account.UserId,
			Name:       account.Name,
			Type:       account.Type,
			Currency:   account.CurrencyCode,
			Actual:     actual.StringFixed(9),
			Expected:   expectedBalance.StringFixed(9),
			Difference: actual.Sub(expectedBalance).StringFixed(9),
		})
	}

	report.Passed = len(report.Differences) == 0 && len(report.Issues) == 0
	return report
}

func validTransactionType(transactionType int) bool {
	switch transactionType {
	case utils.TransactionTypeExpense,
		utils.TransactionTypeIncome,
		utils.TransactionTypeTransfer,
		utils.TransactionTypeOpeningBalance:
		return true
	default:
		return false
	}
}
