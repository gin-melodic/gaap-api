// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// AccountTypes is the golang structure for table account_types.
type AccountTypes struct {
	Type      int         `json:"type"      orm:"type"       description:""` //
	Label     string      `json:"label"     orm:"label"      description:""` //
	Color     string      `json:"color"     orm:"color"      description:""` //
	Bg        string      `json:"bg"        orm:"bg"         description:""` //
	Icon      string      `json:"icon"      orm:"icon"       description:""` //
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:""` //
	UpdatedAt *gtime.Time `json:"updatedAt" orm:"updated_at" description:""` //
	DeletedAt *gtime.Time `json:"deletedAt" orm:"deleted_at" description:""` //
}
