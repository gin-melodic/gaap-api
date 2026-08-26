// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// DemoUserBaselines is the golang structure of table demo_user_baselines for DAO operations like Where/Data.
type DemoUserBaselines struct {
	g.Meta                 `orm:"table:demo_user_baselines, do:true"`
	UserId                 any         //
	UserSnapshot           any         //
	AccountsSnapshot       any         //
	TransactionsSnapshot   any         //
	GenerationRunsSnapshot any         //
	LastResetDate          *gtime.Time //
	CreatedAt              *gtime.Time //
	UpdatedAt              *gtime.Time //
}
