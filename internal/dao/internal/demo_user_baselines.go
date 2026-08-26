// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// DemoUserBaselinesDao is the data access object for the table demo_user_baselines.
type DemoUserBaselinesDao struct {
	table    string                   // table is the underlying table name of the DAO.
	group    string                   // group is the database configuration group name of the current DAO.
	columns  DemoUserBaselinesColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler       // handlers for customized model modification.
}

// DemoUserBaselinesColumns defines and stores column names for the table demo_user_baselines.
type DemoUserBaselinesColumns struct {
	UserId                 string //
	UserSnapshot           string //
	AccountsSnapshot       string //
	TransactionsSnapshot   string //
	GenerationRunsSnapshot string //
	LastResetDate          string //
	CreatedAt              string //
	UpdatedAt              string //
}

// demoUserBaselinesColumns holds the columns for the table demo_user_baselines.
var demoUserBaselinesColumns = DemoUserBaselinesColumns{
	UserId:                 "user_id",
	UserSnapshot:           "user_snapshot",
	AccountsSnapshot:       "accounts_snapshot",
	TransactionsSnapshot:   "transactions_snapshot",
	GenerationRunsSnapshot: "generation_runs_snapshot",
	LastResetDate:          "last_reset_date",
	CreatedAt:              "created_at",
	UpdatedAt:              "updated_at",
}

// NewDemoUserBaselinesDao creates and returns a new DAO object for table data access.
func NewDemoUserBaselinesDao(handlers ...gdb.ModelHandler) *DemoUserBaselinesDao {
	return &DemoUserBaselinesDao{
		group:    "default",
		table:    "demo_user_baselines",
		columns:  demoUserBaselinesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *DemoUserBaselinesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *DemoUserBaselinesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *DemoUserBaselinesDao) Columns() DemoUserBaselinesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *DemoUserBaselinesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *DemoUserBaselinesDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *DemoUserBaselinesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
