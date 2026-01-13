// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// Transactions is the golang structure for table transactions.
type Transactions struct {
	Id             uuid.UUID   `json:"id"             orm:"id"              description:""` //
	UserId         uuid.UUID   `json:"userId"         orm:"user_id"         description:""` //
	Date           *gtime.Time `json:"date"           orm:"date"            description:""` //
	FromAccountId  uuid.UUID   `json:"fromAccountId"  orm:"from_account_id" description:""` //
	ToAccountId    uuid.UUID   `json:"toAccountId"    orm:"to_account_id"   description:""` //
	CurrencyCode   string      `json:"currencyCode"   orm:"currency_code"   description:""` //
	BalanceUnits   int64       `json:"balanceUnits"   orm:"balance_units"   description:""` //
	BalanceNanos   int         `json:"balanceNanos"   orm:"balance_nanos"   description:""` //
	BalanceDecimal float64     `json:"balanceDecimal" orm:"balance_decimal" description:""` //
	Note           string      `json:"note"           orm:"note"            description:""` //
	Type           int         `json:"type"           orm:"type"            description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""` //
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"      description:""` //
	DeletedAt      *gtime.Time `json:"deletedAt"      orm:"deleted_at"      description:""` //
}
