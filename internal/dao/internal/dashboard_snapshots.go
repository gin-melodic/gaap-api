// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// DashboardSnapshotsDao is the data access object for the table dashboard_snapshots.
type DashboardSnapshotsDao struct {
	table    string                    // table is the underlying table name of the DAO.
	group    string                    // group is the database configuration group name of the current DAO.
	columns  DashboardSnapshotsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler        // handlers for customized model modification.
}

// DashboardSnapshotsColumns defines and stores column names for the table dashboard_snapshots.
type DashboardSnapshotsColumns struct {
	Id           string //
	UserId       string //
	SnapshotType string //
	SnapshotKey  string //
	Data         string //
	CreatedAt    string //
	UpdatedAt    string //
}

// dashboardSnapshotsColumns holds the columns for the table dashboard_snapshots.
var dashboardSnapshotsColumns = DashboardSnapshotsColumns{
	Id:           "id",
	UserId:       "user_id",
	SnapshotType: "snapshot_type",
	SnapshotKey:  "snapshot_key",
	Data:         "data",
	CreatedAt:    "created_at",
	UpdatedAt:    "updated_at",
}

// NewDashboardSnapshotsDao creates and returns a new DAO object for table data access.
func NewDashboardSnapshotsDao(handlers ...gdb.ModelHandler) *DashboardSnapshotsDao {
	return &DashboardSnapshotsDao{
		group:    "default",
		table:    "dashboard_snapshots",
		columns:  dashboardSnapshotsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *DashboardSnapshotsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *DashboardSnapshotsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *DashboardSnapshotsDao) Columns() DashboardSnapshotsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *DashboardSnapshotsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *DashboardSnapshotsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *DashboardSnapshotsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
