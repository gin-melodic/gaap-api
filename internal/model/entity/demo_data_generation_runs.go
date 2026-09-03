// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// DemoDataGenerationRuns is the golang structure for table demo_data_generation_runs.
type DemoDataGenerationRuns struct {
	UserId         uuid.UUID   `json:"userId"         orm:"user_id"         description:""` //
	BusinessDate   *gtime.Time `json:"businessDate"   orm:"business_date"   description:""` //
	GeneratedCount int         `json:"generatedCount" orm:"generated_count" description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""` //
}
