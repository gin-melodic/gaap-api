// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// DemoDataGenerationRunsDao is the data access object for the table demo_data_generation_runs.
type DemoDataGenerationRunsDao struct {
	table    string                        // table is the underlying table name of the DAO.
	group    string                        // group is the database configuration group name of the current DAO.
	columns  DemoDataGenerationRunsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler            // handlers for customized model modification.
}

// DemoDataGenerationRunsColumns defines and stores column names for the table demo_data_generation_runs.
type DemoDataGenerationRunsColumns struct {
	UserId         string //
	BusinessDate   string //
	GeneratedCount string //
	CreatedAt      string //
}

// demoDataGenerationRunsColumns holds the columns for the table demo_data_generation_runs.
var demoDataGenerationRunsColumns = DemoDataGenerationRunsColumns{
	UserId:         "user_id",
	BusinessDate:   "business_date",
	GeneratedCount: "generated_count",
	CreatedAt:      "created_at",
}

// NewDemoDataGenerationRunsDao creates and returns a new DAO object for table data access.
func NewDemoDataGenerationRunsDao(handlers ...gdb.ModelHandler) *DemoDataGenerationRunsDao {
	return &DemoDataGenerationRunsDao{
		group:    "default",
		table:    "demo_data_generation_runs",
		columns:  demoDataGenerationRunsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *DemoDataGenerationRunsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *DemoDataGenerationRunsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *DemoDataGenerationRunsDao) Columns() DemoDataGenerationRunsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *DemoDataGenerationRunsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *DemoDataGenerationRunsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *DemoDataGenerationRunsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
