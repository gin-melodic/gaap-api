// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// AccountsDao is the data access object for the table accounts.
type AccountsDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  AccountsColumns    // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// AccountsColumns defines and stores column names for the table accounts.
type AccountsColumns struct {
	Id             string //
	UserId         string //
	ParentId       string //
	Name           string //
	Type           string //
	IsGroup        string //
	Balance        string //
	Currency       string //
	DefaultChildId string //
	Date           string //
	Number         string //
	Remarks        string //
	CreatedAt      string //
	UpdatedAt      string //
	DeletedAt      string //
}

// accountsColumns holds the columns for the table accounts.
var accountsColumns = AccountsColumns{
	Id:             "id",
	UserId:         "user_id",
	ParentId:       "parent_id",
	Name:           "name",
	Type:           "type",
	IsGroup:        "is_group",
	Balance:        "balance",
	Currency:       "currency",
	DefaultChildId: "default_child_id",
	Date:           "date",
	Number:         "number",
	Remarks:        "remarks",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
	DeletedAt:      "deleted_at",
}

// NewAccountsDao creates and returns a new DAO object for table data access.
func NewAccountsDao(handlers ...gdb.ModelHandler) *AccountsDao {
	return &AccountsDao{
		group:    "default",
		table:    "accounts",
		columns:  accountsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *AccountsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *AccountsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *AccountsDao) Columns() AccountsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *AccountsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *AccountsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *AccountsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
