// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// MigrationMappings is the golang structure of table migration_mappings for DAO operations like Where/Data.
type MigrationMappings struct {
	g.Meta    `orm:"table:migration_mappings, do:true"`
	Id        any         //
	TaskId    any         //
	TableName any         //
	RecordId  any         //
	FieldName any         //
	OldValue  any         //
	NewValue  any         //
	Applied   any         //
	CreatedAt *gtime.Time //
}
