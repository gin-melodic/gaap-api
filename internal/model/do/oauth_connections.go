// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// OauthConnections is the golang structure of table oauth_connections for DAO operations like Where/Data.
type OauthConnections struct {
	g.Meta         `orm:"table:oauth_connections, do:true"`
	Id             interface{} //
	UserId         interface{} //
	Provider       interface{} //
	ProviderUserId interface{} //
	AccessToken    interface{} //
	RefreshToken   interface{} //
	CreatedAt      *gtime.Time //
	UpdatedAt      *gtime.Time //
}
