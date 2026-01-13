// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Accounts is the golang structure of table accounts for DAO operations like Where/Data.
type Accounts struct {
	g.Meta         `orm:"table:accounts, do:true"`
	Id             any         //
	UserId         any         //
	ParentId       any         //
	Name           any         //
	Type           any         //
	IsGroup        any         //
	CurrencyCode   any         //
	BalanceUnits   any         //
	BalanceNanos   any         //
	BalanceDecimal any         //
	DefaultChildId any         //
	Date           *gtime.Time //
	Number         any         //
	Remarks        any         //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
	DeletedAt      *gtime.Time //
}
