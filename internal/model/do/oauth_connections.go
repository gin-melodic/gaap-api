// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// OauthConnections is the golang structure of table oauth_connections for DAO operations like Where/Data.
type OauthConnections struct {
	g.Meta         `orm:"table:oauth_connections, do:true"`
	Id             any         //
	UserId         any         //
	Provider       any         //
	ProviderUserId any         //
	AccessToken    any         //
	RefreshToken   any         //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
}
