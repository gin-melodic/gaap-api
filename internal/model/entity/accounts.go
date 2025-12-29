// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Accounts is the golang structure for table accounts.
type Accounts struct {
	Id             string      `json:"id"             orm:"id"               description:""` //
	UserId         string      `json:"userId"         orm:"user_id"          description:""` //
	ParentId       string      `json:"parentId"       orm:"parent_id"        description:""` //
	Name           string      `json:"name"           orm:"name"             description:""` //
	Type           string      `json:"type"           orm:"type"             description:""` //
	IsGroup        bool        `json:"isGroup"        orm:"is_group"         description:""` //
	Balance        float64     `json:"balance"        orm:"balance"          description:""` //
	Currency       string      `json:"currency"       orm:"currency"         description:""` //
	DefaultChildId string      `json:"defaultChildId" orm:"default_child_id" description:""` //
	Date           *gtime.Time `json:"date"           orm:"date"             description:""` //
	Number         string      `json:"number"         orm:"number"           description:""` //
	Remarks        string      `json:"remarks"        orm:"remarks"          description:""` //
	CreatedAt      *gtime.Time `json:"createdAt"      orm:"created_at"       description:""` //
	UpdatedAt      *gtime.Time `json:"updatedAt"      orm:"updated_at"       description:""` //
	DeletedAt      *gtime.Time `json:"deletedAt"      orm:"deleted_at"       description:""` //
}
