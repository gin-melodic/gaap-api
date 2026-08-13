package account

import (
	"context"
	"strings"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/google/uuid"
)

func validateAccountType(accountType int) error {
	if accountType < utils.AccountTypeAsset || accountType > utils.AccountTypeEquity {
		return gerror.New("invalid account type")
	}
	return nil
}

func resolveUserBaseCurrency(ctx context.Context, tx gdb.TX, userId uuid.UUID, requested string) (string, error) {
	var user entity.Users
	err := tx.Model(dao.Users.Table()).
		Fields(dao.Users.Columns().MainCurrency).
		Where(dao.Users.Columns().Id, userId).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&user)
	if err != nil {
		return "", gerror.Wrap(err, "failed to load user base currency")
	}
	baseCurrency := strings.ToUpper(strings.TrimSpace(user.MainCurrency))
	if baseCurrency == "" {
		return "", gerror.New("user base currency is not configured")
	}
	requested = strings.ToUpper(strings.TrimSpace(requested))
	if requested != "" && requested != baseCurrency {
		return "", gerror.New("account currency must match user base currency")
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
