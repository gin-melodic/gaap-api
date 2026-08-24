// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// DashboardSnapshots is the golang structure for table dashboard_snapshots.
type DashboardSnapshots struct {
	Id           uuid.UUID   `json:"id"           orm:"id"            description:""` //
	UserId       uuid.UUID   `json:"userId"       orm:"user_id"       description:""` //
	SnapshotType string      `json:"snapshotType" orm:"snapshot_type" description:""` //
	SnapshotKey  string      `json:"snapshotKey"  orm:"snapshot_key"  description:""` //
	Data         string      `json:"data"         orm:"data"          description:""` //
	CreatedAt    *gtime.Time `json:"createdAt"    orm:"created_at"    description:""` //
	UpdatedAt    *gtime.Time `json:"updatedAt"    orm:"updated_at"    description:""` //
}
