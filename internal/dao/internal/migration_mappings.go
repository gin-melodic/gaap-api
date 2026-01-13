// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// MigrationMappingsDao is the data access object for the table migration_mappings.
type MigrationMappingsDao struct {
	table    string                   // table is the underlying table name of the DAO.
	group    string                   // group is the database configuration group name of the current DAO.
	columns  MigrationMappingsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler       // handlers for customized model modification.
}

// MigrationMappingsColumns defines and stores column names for the table migration_mappings.
type MigrationMappingsColumns struct {
	Id        string //
	TaskId    string //
	TableName string //
	RecordId  string //
	FieldName string //
	OldValue  string //
	NewValue  string //
	Applied   string //
	CreatedAt string //
}

// migrationMappingsColumns holds the columns for the table migration_mappings.
var migrationMappingsColumns = MigrationMappingsColumns{
	Id:        "id",
	TaskId:    "task_id",
	TableName: "table_name",
	RecordId:  "record_id",
	FieldName: "field_name",
	OldValue:  "old_value",
	NewValue:  "new_value",
	Applied:   "applied",
	CreatedAt: "created_at",
}

// NewMigrationMappingsDao creates and returns a new DAO object for table data access.
func NewMigrationMappingsDao(handlers ...gdb.ModelHandler) *MigrationMappingsDao {
	return &MigrationMappingsDao{
		group:    "default",
		table:    "migration_mappings",
		columns:  migrationMappingsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *MigrationMappingsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *MigrationMappingsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *MigrationMappingsDao) Columns() MigrationMappingsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *MigrationMappingsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *MigrationMappingsDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *MigrationMappingsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
