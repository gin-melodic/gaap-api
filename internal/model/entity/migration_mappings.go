// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// MigrationMappings is the golang structure for table migration_mappings.
type MigrationMappings struct {
	Id        uuid.UUID   `json:"id"        orm:"id"         description:""` //
	TaskId    uuid.UUID   `json:"taskId"    orm:"task_id"    description:""` //
	TableName string      `json:"tableName" orm:"table_name" description:""` //
	RecordId  uuid.UUID   `json:"recordId"  orm:"record_id"  description:""` //
	FieldName string      `json:"fieldName" orm:"field_name" description:""` //
	OldValue  uuid.UUID   `json:"oldValue"  orm:"old_value"  description:""` //
	NewValue  uuid.UUID   `json:"newValue"  orm:"new_value"  description:""` //
	Applied   bool        `json:"applied"   orm:"applied"    description:""` //
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""` //
}
