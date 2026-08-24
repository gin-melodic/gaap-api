package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type DemoDataGenerationRuns struct {
	g.Meta         `orm:"table:demo_data_generation_runs, do:true"`
	UserId         any
	BusinessDate   any
	GeneratedCount any
	CreatedAt      *gtime.Time
}
