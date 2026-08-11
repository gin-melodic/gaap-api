// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// DashboardSnapshots is the golang structure of table dashboard_snapshots for DAO operations like Where/Data.
type DashboardSnapshots struct {
	g.Meta       `orm:"table:dashboard_snapshots, do:true"`
	Id           any         //
	UserId       any         //
	SnapshotType any         //
	SnapshotKey  any         //
	Data         any         //
	CreatedAt    *gtime.Time //
	UpdatedAt    *gtime.Time //
}
