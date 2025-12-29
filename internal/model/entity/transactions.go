// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Transactions is the golang structure for table transactions.
type Transactions struct {
	Id            string      `json:"id"            orm:"id"              description:""` //
	UserId        string      `json:"userId"        orm:"user_id"         description:""` //
	Date          *gtime.Time `json:"date"          orm:"date"            description:""` //
	FromAccountId string      `json:"fromAccountId" orm:"from_account_id" description:""` //
	ToAccountId   string      `json:"toAccountId"   orm:"to_account_id"   description:""` //
	Amount        float64     `json:"amount"        orm:"amount"          description:""` //
	Currency      string      `json:"currency"      orm:"currency"        description:""` //
	Note          string      `json:"note"          orm:"note"            description:""` //
	Type          string      `json:"type"          orm:"type"            description:""` //
	CreatedAt     *gtime.Time `json:"createdAt"     orm:"created_at"      description:""` //
	UpdatedAt     *gtime.Time `json:"updatedAt"     orm:"updated_at"      description:""` //
	DeletedAt     *gtime.Time `json:"deletedAt"     orm:"deleted_at"      description:""` //
}
