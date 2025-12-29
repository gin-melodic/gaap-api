package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type Account struct {
	Id             string      `json:"id" v:"required|max-length:50|regex:^[a-zA-Z0-9_-]+$"`
	ParentId       *string     `json:"parentId" v:"max-length:50|regex:^[a-zA-Z0-9_-]+$"`
	Name           string      `json:"name" v:"required|max-length:100"`
	Type           string      `json:"type" v:"required|in:ASSET,LIABILITY,INCOME,EXPENSE"`
	IsGroup        bool        `json:"isGroup"`
	Balance        float64     `json:"balance" v:"required|min:-1000000000000|max:1000000000000"`
	Currency       string      `json:"currency" v:"required"`
	DefaultChildId *string     `json:"defaultChildId" v:"max-length:50"`
	Date           string      `json:"date" v:"date"`
	Number         string      `json:"number" v:"max-length:50"`
	Remarks        string      `json:"remarks" v:"max-length:500"`
	CreatedAt      *gtime.Time `json:"created_at"`
	UpdatedAt      *gtime.Time `json:"updated_at"`
}

type AccountInput struct {
	ParentId       *string `json:"parentId" v:"max-length:50"`
	Name           string  `json:"name" v:"required|max-length:100"`
	Type           string  `json:"type" v:"required|in:ASSET,LIABILITY,INCOME,EXPENSE"`
	IsGroup        bool    `json:"isGroup"`
	Balance        float64 `json:"balance" v:"min:-1000000000000|max:1000000000000"`
	Currency       string  `json:"currency" v:"required"`
	DefaultChildId *string `json:"defaultChildId" v:"max-length:50"`
	Date           string  `json:"date" v:"date"`
	Number         string  `json:"number" v:"max-length:50"`
	Remarks        string  `json:"remarks" v:"max-length:500"`
}

type AccountQuery struct {
	Page     int    `json:"page" v:"min:1" d:"1"`
	Limit    int    `json:"limit" v:"min:1|max:100" d:"20"`
	Type     string `json:"type" v:"in:ASSET,LIABILITY,INCOME,EXPENSE"`
	ParentId string `json:"parentId"`
}

type ListAccountsReq struct {
	g.Meta `path:"/accounts" tags:"Accounts" method:"get" summary:"List all accounts"`
	AccountQuery
}

type ListAccountsRes struct {
	g.Meta `mime:"application/json"`
	common.PaginatedResponse
	*common.BaseResponse
	Data []Account `json:"data"`
}

type CreateAccountReq struct {
	g.Meta `path:"/accounts" tags:"Accounts" method:"post" summary:"Create a new account"`
	*AccountInput
}

type CreateAccountRes struct {
	g.Meta `mime:"application/json"`
	*Account
	*common.BaseResponse
}

type GetAccountReq struct {
	g.Meta `path:"/accounts/{id}" tags:"Accounts" method:"get" summary:"Get account details"`
	Id     string `json:"id" v:"required"`
}

type GetAccountRes struct {
	g.Meta `mime:"application/json"`
	*Account
	*common.BaseResponse
}

type UpdateAccountReq struct {
	g.Meta `path:"/accounts/{id}" tags:"Accounts" method:"put" summary:"Update account"`
	Id     string `json:"id" v:"required"`
	*AccountInput
}

type UpdateAccountRes struct {
	g.Meta `mime:"application/json"`
	*Account
	*common.BaseResponse
}

type DeleteAccountReq struct {
	g.Meta           `path:"/accounts/{id}" tags:"Accounts" method:"delete" summary:"Delete account with optional migration"`
	Id               string            `json:"id" v:"required"`
	MigrationTargets map[string]string `json:"migrationTargets"` // currency -> targetAccountId
}

type DeleteAccountRes struct {
	g.Meta `mime:"application/json"`
	TaskId string `json:"taskId,omitempty"` // Task ID for async migration
	*common.BaseResponse
}
