// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Tasks is the golang structure for table tasks.
type Tasks struct {
	Id             string      `json:"id"             orm:"id"              description:""`
	UserId         string      `json:"userId"         orm:"user_id"         description:""`
	Type           string      `json:"type"           orm:"type"            description:""`
	Status         string      `json:"status"         orm:"status"          description:""`
	Payload        string      `json:"payload"        orm:"payload"         description:"JSONB"`
	Result         string      `json:"result"         orm:"result"          description:"JSONB"`
	Progress       int         `json:"progress"       orm:"progress"        description:""`
	TotalItems     int         `json:"totalItems"     orm:"total_items"     description:""`
	ProcessedItems int         `json:"processedItems" orm:"processed_items" description:""`
	StartedAt      *gtime.Time `json:"startedAt"      orm:"started_at"      description:""`
	CompletedAt    *gtime.Time `json:"completedAt"    orm:"completed_at"    description:""`
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"      description:""`
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"      description:""`
}
