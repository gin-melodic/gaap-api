package utils

import (
	"context"
	"strings"

	"gaap-api/internal/dao"

	"github.com/gogf/gf/v2/errors/gerror"
)

// NormalizeCurrency uppercases and trims a currency code.
func NormalizeCurrency(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// CurrencyExists reports whether the given currency code is a supported
// currency that has not been soft-deleted.
func CurrencyExists(ctx context.Context, code string) (bool, error) {
	code = NormalizeCurrency(code)
	if code == "" {
		return false, nil
	}
	count, err := dao.Currencies.Ctx(ctx).
		Where(dao.Currencies.Columns().Code, code).
		WhereNull(dao.Currencies.Columns().DeletedAt).
		Count()
	if err != nil {
		return false, gerror.Wrap(err, "failed to validate currency")
	}
	return count > 0, nil
}
