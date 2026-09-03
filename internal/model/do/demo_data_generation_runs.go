// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// DemoDataGenerationRuns is the golang structure of table demo_data_generation_runs for DAO operations like Where/Data.
type DemoDataGenerationRuns struct {
	g.Meta         `orm:"table:demo_data_generation_runs, do:true"`
	UserId         any         //
	BusinessDate   *gtime.Time //
	GeneratedCount any         //
	CreatedAt      *gtime.Time //
}
