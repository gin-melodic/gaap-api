// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// Accounts is the golang structure for table accounts.
type Accounts struct {
	Id              uuid.UUID   `json:"id"              orm:"id"                description:""` //
	UserId          uuid.UUID   `json:"userId"          orm:"user_id"           description:""` //
	ParentId        uuid.UUID   `json:"parentId"        orm:"parent_id"         description:""` //
	Name            string      `json:"name"            orm:"name"              description:""` //
	Type            int         `json:"type"            orm:"type"              description:""` //
	IsGroup         bool        `json:"isGroup"         orm:"is_group"          description:""` //
	CurrencyCode    string      `json:"currencyCode"    orm:"currency_code"     description:""` //
	BalanceUnits    int64       `json:"balanceUnits"    orm:"balance_units"     description:""` //
	BalanceNanos    int         `json:"balanceNanos"    orm:"balance_nanos"     description:""` //
	BalanceDecimal  float64     `json:"balanceDecimal"  orm:"balance_decimal"   description:""` //
	DefaultChildId  uuid.UUID   `json:"defaultChildId"  orm:"default_child_id"  description:""` //
	EquityAccountId uuid.UUID   `json:"equityAccountId" orm:"equity_account_id" description:""` //
	Date            *gtime.Time `json:"date"            orm:"date"              description:""` //
	Number          string      `json:"number"          orm:"number"            description:""` //
	Remarks         string      `json:"remarks"         orm:"remarks"           description:""` //
	CreatedAt       *gtime.Time `json:"createdAt"       orm:"created_at"        description:""` //
	UpdatedAt       *gtime.Time `json:"updatedAt"       orm:"updated_at"        description:""` //
	DeletedAt       *gtime.Time `json:"deletedAt"       orm:"deleted_at"        description:""` //
}
