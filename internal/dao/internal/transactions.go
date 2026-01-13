// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TransactionsDao is the data access object for the table transactions.
type TransactionsDao struct {
	table    string              // table is the underlying table name of the DAO.
	group    string              // group is the database configuration group name of the current DAO.
	columns  TransactionsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler  // handlers for customized model modification.
}

// TransactionsColumns defines and stores column names for the table transactions.
type TransactionsColumns struct {
	Id             string //
	UserId         string //
	Date           string //
	FromAccountId  string //
	ToAccountId    string //
	CurrencyCode   string //
	BalanceUnits   string //
	BalanceNanos   string //
	BalanceDecimal string //
	Note           string //
	Type           string //
	CreatedAt      string //
	UpdatedAt      string //
	DeletedAt      string //
}

// transactionsColumns holds the columns for the table transactions.
var transactionsColumns = TransactionsColumns{
	Id:             "id",
	UserId:         "user_id",
	Date:           "date",
	FromAccountId:  "from_account_id",
	ToAccountId:    "to_account_id",
	CurrencyCode:   "currency_code",
	BalanceUnits:   "balance_units",
	BalanceNanos:   "balance_nanos",
	BalanceDecimal: "balance_decimal",
	Note:           "note",
	Type:           "type",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
	DeletedAt:      "deleted_at",
}

// NewTransactionsDao creates and returns a new DAO object for table data access.
func NewTransactionsDao(handlers ...gdb.ModelHandler) *TransactionsDao {
	return &TransactionsDao{
		group:    "default",
		table:    "transactions",
		columns:  transactionsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TransactionsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TransactionsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TransactionsDao) Columns() TransactionsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TransactionsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TransactionsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *TransactionsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
