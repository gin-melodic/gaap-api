// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Accounts is the golang structure of table accounts for DAO operations like Where/Data.
type Accounts struct {
	g.Meta         `orm:"table:accounts, do:true"`
	Id             interface{} //
	UserId         interface{} //
	ParentId       interface{} //
	Name           interface{} //
	Type           interface{} //
	IsGroup        interface{} //
	Balance        interface{} //
	Currency       interface{} //
	DefaultChildId interface{} //
	Date           *gtime.Time //
	Number         interface{} //
	Remarks        interface{} //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
	DeletedAt      *gtime.Time //
}
