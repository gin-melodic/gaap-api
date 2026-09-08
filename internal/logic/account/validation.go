package account

import (
	"context"
	"strings"
	"unicode/utf8"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/google/uuid"
)

const (
	maxAccountNameLength    = 100
	maxAccountNumberLength  = 50
	maxAccountRemarksLength = 500
)

func validateAccountMetadata(name, number, remarks string) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return gerror.New("account name is required")
	}
	if utf8.RuneCountInString(trimmedName) > maxAccountNameLength {
		return gerror.New("account name must not exceed 100 characters")
	}
	if utf8.RuneCountInString(number) > maxAccountNumberLength {
		return gerror.New("account number must not exceed 50 characters")
	}
	if utf8.RuneCountInString(remarks) > maxAccountRemarksLength {
		return gerror.New("account remarks must not exceed 500 characters")
	}
	return nil
}

func validateAccountType(accountType int) error {
	if accountType < utils.AccountTypeAsset || accountType > utils.AccountTypeEquity {
		return gerror.New("invalid account type")
	}
	return nil
}

// resolveAccountCurrency returns the currency for a new/updated account.
// An empty requested currency defaults to the user's base currency; a non-empty
// value is accepted only when it is a supported (non-deleted) currency, which
// enables multi-currency standalone accounts.
func resolveAccountCurrency(ctx context.Context, tx gdb.TX, userId uuid.UUID, requested string) (string, error) {
	baseCurrency, err := loadUserBaseCurrency(ctx, tx, userId)
	if err != nil {
		return "", err
	}
	requested = utils.NormalizeCurrency(requested)
	if requested == "" {
		return baseCurrency, nil
	}
	exists, err := utils.CurrencyExists(ctx, requested)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", gerror.Newf("unsupported currency %q", requested)
	}
	return requested, nil
}

// loadUserBaseCurrency loads and normalizes the user's configured base currency.
func loadUserBaseCurrency(ctx context.Context, tx gdb.TX, userId uuid.UUID) (string, error) {
	var user entity.Users
	err := tx.Model(dao.Users.Table()).
		Fields(dao.Users.Columns().MainCurrency).
		Where(dao.Users.Columns().Id, userId).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&user)
	if err != nil {
		return "", gerror.Wrap(err, "failed to load user base currency")
	}
	baseCurrency := utils.NormalizeCurrency(user.MainCurrency)
	if baseCurrency == "" {
		return "", gerror.New("user base currency is not configured")
	}
	return baseCurrency, nil
}

func validateAccountHierarchyAccess(plan int, isGroup bool, parentId uuid.UUID) error {
	if !isGroup && parentId == uuid.Nil {
		return nil
	}
	if plan != utils.UserLevelPro {
		return gerror.New("account groups and child accounts require a Pro plan")
	}
	return nil
}

func validateUserAccountHierarchyAccess(ctx context.Context, tx gdb.TX, userId uuid.UUID, isGroup bool, parentId uuid.UUID) error {
	if !isGroup && parentId == uuid.Nil {
		return nil
	}

	var user entity.Users
	if err := tx.Model(dao.Users.Table()).
		Fields(dao.Users.Columns().Plan).
		Where(dao.Users.Columns().Id, userId).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&user); err != nil {
		return gerror.Wrap(err, "failed to load user plan")
	}
	return validateAccountHierarchyAccess(user.Plan, isGroup, parentId)
}

func validateParentAccount(ctx context.Context, tx gdb.TX, parentId, userId uuid.UUID, currency string) error {
	if parentId == uuid.Nil {
		return nil
	}
	var parent entity.Accounts
	err := tx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().Id, parentId).
		Where(dao.Accounts.Columns().UserId, userId).
		WhereNull(dao.Accounts.Columns().DeletedAt).
		LockUpdate().
		Scan(&parent)
	if err != nil {
		return gerror.Wrap(err, "failed to load parent account")
	}
	if parent.Id == uuid.Nil || !parent.IsGroup {
		return gerror.New("parent account must be an active account group owned by the user")
	}
	if !strings.EqualFold(parent.CurrencyCode, currency) {
		return gerror.New("parent account currency mismatch")
	}
	return nil
}
