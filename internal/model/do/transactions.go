// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Transactions is the golang structure of table transactions for DAO operations like Where/Data.
type Transactions struct {
	g.Meta         `orm:"table:transactions, do:true"`
	Id             any         //
	UserId         any         //
	Date           *gtime.Time //
	FromAccountId  any         //
	ToAccountId    any         //
	CurrencyCode   any         //
	BalanceUnits   any         //
	BalanceNanos   any         //
	BalanceDecimal any         //
	Note           any         //
	Type           any         //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
	DeletedAt      *gtime.Time //
}
