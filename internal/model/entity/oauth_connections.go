// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// OauthConnections is the golang structure for table oauth_connections.
type OauthConnections struct {
	Id             string      `json:"id"             orm:"id"               description:""` //
	UserId         string      `json:"userId"         orm:"user_id"          description:""` //
	Provider       string      `json:"provider"       orm:"provider"         description:""` //
	ProviderUserId string      `json:"providerUserId" orm:"provider_user_id" description:""` //
	AccessToken    string      `json:"accessToken"    orm:"access_token"     description:""` //
	RefreshToken   string      `json:"refreshToken"   orm:"refresh_token"    description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"       description:""` //
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"       description:""` //
}
