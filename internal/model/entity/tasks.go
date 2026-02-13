// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// Tasks is the golang structure for table tasks.
type Tasks struct {
	Id             uuid.UUID   `json:"id"             orm:"id"              description:""` //
	UserId         uuid.UUID   `json:"userId"         orm:"user_id"         description:""` //
	Type           int         `json:"type"           orm:"type"            description:""` //
	Status         int         `json:"status"         orm:"status"          description:""` //
	Payload        string      `json:"payload"        orm:"payload"         description:""` //
	Result         string      `json:"result"         orm:"result"          description:""` //
	Progress       int         `json:"progress"       orm:"progress"        description:""` //
	TotalItems     int         `json:"totalItems"     orm:"total_items"     description:""` //
	ProcessedItems int         `json:"processedItems" orm:"processed_items" description:""` //
	StartedAt      *gtime.Time `json:"startedAt"      orm:"started_at"      description:""` //
	CompletedAt    *gtime.Time `json:"completedAt"    orm:"completed_at"    description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""` //
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"      description:""` //
}
