// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// DemoUserBaselines is the golang structure for table demo_user_baselines.
type DemoUserBaselines struct {
	UserId                 uuid.UUID   `json:"userId"                 orm:"user_id"                  description:""` //
	UserSnapshot           string      `json:"userSnapshot"           orm:"user_snapshot"            description:""` //
	AccountsSnapshot       string      `json:"accountsSnapshot"       orm:"accounts_snapshot"        description:""` //
	TransactionsSnapshot   string      `json:"transactionsSnapshot"   orm:"transactions_snapshot"    description:""` //
	GenerationRunsSnapshot string      `json:"generationRunsSnapshot" orm:"generation_runs_snapshot" description:""` //
	LastResetDate          *gtime.Time `json:"lastResetDate"          orm:"last_reset_date"          description:""` //
	CreatedAt              *gtime.Time `json:"createdAt"              orm:"created_at"               description:""` //
	UpdatedAt              *gtime.Time `json:"updatedAt"              orm:"updated_at"               description:""` //
}
