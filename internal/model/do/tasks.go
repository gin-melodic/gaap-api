// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Tasks is the golang structure of table tasks for DAO operations like Where/Data.
type Tasks struct {
	g.Meta         `orm:"table:tasks, do:true"`
	Id             any         //
	UserId         any         //
	Type           any         //
	Status         any         //
	Payload        any         //
	Result         any         //
	Progress       any         //
	TotalItems     any         //
	ProcessedItems any         //
	StartedAt      *gtime.Time //
	CompletedAt    *gtime.Time //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
}
