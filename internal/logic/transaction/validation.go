package transaction

import (
	"context"
	"sort"
	"strings"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func validateMoney(units int64, nanos int, allowSigned bool) error {
	if nanos < -999_999_999 || nanos > 999_999_999 {
		return gerror.New("amount nanos out of range")
	}
	if (units > 0 && nanos < 0) || (units < 0 && nanos > 0) {
		return gerror.New("amount units and nanos must use the same sign")
	}
	amount := decimal.NewFromInt(units).Add(decimal.New(int64(nanos), -9))
	if amount.IsZero() {
		return gerror.New("amount must not be zero")
	}
	if !allowSigned && !amount.IsPositive() {
		return gerror.New("amount must be greater than zero")
	}
	return nil
}

func validateTransactionAccounts(
	ctx context.Context,
	tx gdb.TX,
	userId string,
	in *model.TransactionCreateInput,
) error {
	if in.FromAccountId == in.ToAccountId {
		return gerror.New("source and destination accounts must differ")
	}
	if err := validateMoney(in.BalanceUnits, in.BalanceNanos, in.Type == utils.TransactionTypeOpeningBalance); err != nil {
		return err
	}

	var user entity.Users
	err := tx.Model(dao.Users.Table()).
		Fields(dao.Users.Columns().MainCurrency).
		Where(dao.Users.Columns().Id, userId).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&user)
	if err != nil {
		return gerror.Wrap(err, "failed to load user currency")
	}
	if user.MainCurrency == "" {
		return gerror.New("user base currency is not configured")
	}

	currency := strings.ToUpper(strings.TrimSpace(in.CurrencyCode))
	if currency != strings.ToUpper(user.MainCurrency) {
		return gerror.New("transaction currency must match user base currency")
	}
	in.CurrencyCode = currency

	accounts, err := lockOwnedAccounts(ctx, tx, userId, []uuid.UUID{in.FromAccountId, in.ToAccountId})
	if err != nil {
		return gerror.Wrap(err, "failed to lock transaction accounts")
	}
	if len(accounts) != 2 {
		return gerror.New("transaction account not found or unauthorized")
	}

	byId := make(map[string]entity.Accounts, len(accounts))
	for _, account := range accounts {
		if account.IsGroup {
			return gerror.New("account groups cannot be used in transactions")
		}
		if strings.ToUpper(account.CurrencyCode) != currency {
			return gerror.New("account currency mismatch")
		}
		byId[account.Id.String()] = account
	}
	from := byId[in.FromAccountId.String()]
	to := byId[in.ToAccountId.String()]

	return validateTransactionAccountTypes(in.Type, from.Type, to.Type)
}

func validateTransactionAccountTypes(transactionType, fromType, toType int) error {
	switch transactionType {
	case utils.TransactionTypeIncome:
		if fromType != utils.AccountTypeIncome || (toType != utils.AccountTypeAsset && toType != utils.AccountTypeLiability) {
			return gerror.New("invalid account types for income transaction")
		}
	case utils.TransactionTypeExpense:
		if (fromType != utils.AccountTypeAsset && fromType != utils.AccountTypeLiability) || toType != utils.AccountTypeExpense {
			return gerror.New("invalid account types for expense transaction")
		}
	case utils.TransactionTypeTransfer:
		if (fromType != utils.AccountTypeAsset && fromType != utils.AccountTypeLiability) ||
			(toType != utils.AccountTypeAsset && toType != utils.AccountTypeLiability) {
			return gerror.New("invalid account types for transfer transaction")
		}
	case utils.TransactionTypeOpeningBalance:
		if fromType != utils.AccountTypeEquity || (toType != utils.AccountTypeAsset && toType != utils.AccountTypeLiability) {
			return gerror.New("invalid account types for opening balance transaction")
		}
	default:
		return gerror.New("invalid transaction type")
	}
	return nil
}

func lockOwnedAccounts(ctx context.Context, tx gdb.TX, userId string, accountIds []uuid.UUID) ([]entity.Accounts, error) {
	unique := make(map[string]struct{}, len(accountIds))
	ids := make([]string, 0, len(accountIds))
	for _, accountId := range accountIds {
		if accountId == uuid.Nil {
			continue
		}
		id := accountId.String()
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var accounts []entity.Accounts
	err := tx.Model(dao.Accounts.Table()).
		WhereIn(dao.Accounts.Columns().Id, ids).
		Where(dao.Accounts.Columns().UserId, userId).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		OrderAsc(dao.Accounts.Columns().Id).
		LockUpdate().
		Scan(&accounts)
	if err != nil {
		return nil, err
	}
	if len(accounts) != len(ids) {
		return nil, gerror.New("transaction account not found or unauthorized")
	}
	return accounts, nil
}
