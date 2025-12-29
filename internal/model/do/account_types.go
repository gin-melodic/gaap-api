// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// AccountTypes is the golang structure of table account_types for DAO operations like Where/Data.
type AccountTypes struct {
	g.Meta    `orm:"table:account_types, do:true"`
	Type      interface{} //
	Label     interface{} //
	Color     interface{} //
	Bg        interface{} //
	Icon      interface{} //
	CreatedAt *gtime.Time //
	UpdatedAt *gtime.Time //
	DeletedAt *gtime.Time //
}
