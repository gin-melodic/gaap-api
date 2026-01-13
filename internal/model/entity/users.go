// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT. Created at 2026-01-08 17:02:34
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// Users is the golang structure for table users.
type Users struct {
	Id               uuid.UUID   `json:"id"               orm:"id"                 description:""` //
	Password         string      `json:"password"         orm:"password"           description:""` //
	Email            string      `json:"email"            orm:"email"              description:""` //
	Nickname         string      `json:"nickname"         orm:"nickname"           description:""` //
	Avatar           string      `json:"avatar"           orm:"avatar"             description:""` //
	Plan             int         `json:"plan"             orm:"plan"               description:""` //
	ThemeId          uuid.UUID   `json:"themeId"          orm:"theme_id"           description:""` //
	MainCurrency     string      `json:"mainCurrency"     orm:"main_currency"      description:""` //
	TwoFactorSecret  string      `json:"twoFactorSecret"  orm:"two_factor_secret"  description:""` //
	TwoFactorEnabled bool        `json:"twoFactorEnabled" orm:"two_factor_enabled" description:""` //
	CreatedAt        *gtime.Time `json:"createdAt"        orm:"created_at"         description:""` //
	UpdatedAt        *gtime.Time `json:"updatedAt"        orm:"updated_at"         description:""` //
	DeletedAt        *gtime.Time `json:"deletedAt"        orm:"deleted_at"         description:""` //
}
