// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Themes is the golang structure of table themes for DAO operations like Where/Data.
type Themes struct {
	g.Meta    `orm:"table:themes, do:true"`
	Id        interface{} //
	Name      interface{} //
	IsDark    interface{} //
	Colors    interface{} //
	CreatedAt *gtime.Time //
	UpdatedAt *gtime.Time //
	DeletedAt *gtime.Time //
}
