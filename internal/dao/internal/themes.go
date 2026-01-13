// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ThemesDao is the data access object for the table themes.
type ThemesDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  ThemesColumns      // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// ThemesColumns defines and stores column names for the table themes.
type ThemesColumns struct {
	Id        string //
	Name      string //
	IsDark    string //
	Colors    string //
	CreatedAt string //
	UpdatedAt string //
	DeletedAt string //
}

// themesColumns holds the columns for the table themes.
var themesColumns = ThemesColumns{
	Id:        "id",
	Name:      "name",
	IsDark:    "is_dark",
	Colors:    "colors",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewThemesDao creates and returns a new DAO object for table data access.
func NewThemesDao(handlers ...gdb.ModelHandler) *ThemesDao {
	return &ThemesDao{
		group:    "default",
		table:    "themes",
		columns:  themesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *ThemesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *ThemesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *ThemesDao) Columns() ThemesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *ThemesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *ThemesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *ThemesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
