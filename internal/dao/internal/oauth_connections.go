// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// OauthConnectionsDao is the data access object for the table oauth_connections.
type OauthConnectionsDao struct {
	table    string                  // table is the underlying table name of the DAO.
	group    string                  // group is the database configuration group name of the current DAO.
	columns  OauthConnectionsColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler      // handlers for customized model modification.
}

// OauthConnectionsColumns defines and stores column names for the table oauth_connections.
type OauthConnectionsColumns struct {
	Id             string //
	UserId         string //
	Provider       string //
	ProviderUserId string //
	AccessToken    string //
	RefreshToken   string //
	CreatedAt      string //
	UpdatedAt      string //
}

// oauthConnectionsColumns holds the columns for the table oauth_connections.
var oauthConnectionsColumns = OauthConnectionsColumns{
	Id:             "id",
	UserId:         "user_id",
	Provider:       "provider",
	ProviderUserId: "provider_user_id",
	AccessToken:    "access_token",
	RefreshToken:   "refresh_token",
	CreatedAt:      "created_at",
	UpdatedAt:      "updated_at",
}

// NewOauthConnectionsDao creates and returns a new DAO object for table data access.
func NewOauthConnectionsDao(handlers ...gdb.ModelHandler) *OauthConnectionsDao {
	return &OauthConnectionsDao{
		group:    "default",
		table:    "oauth_connections",
		columns:  oauthConnectionsColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *OauthConnectionsDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *OauthConnectionsDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *OauthConnectionsDao) Columns() OauthConnectionsColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *OauthConnectionsDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *OauthConnectionsDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *OauthConnectionsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
