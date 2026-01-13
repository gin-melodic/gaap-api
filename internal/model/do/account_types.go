// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// AccountTypes is the golang structure of table account_types for DAO operations like Where/Data.
type AccountTypes struct {
	g.Meta    `orm:"table:account_types, do:true"`
	Type      any         //
	Label     any         //
	Color     any         //
	Bg        any         //
	Icon      any         //
	CreatedAt *gtime.Time //
	UpdatedAt *gtime.Time //
	DeletedAt *gtime.Time //
}
