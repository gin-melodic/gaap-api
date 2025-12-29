package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type Transaction struct {
	Id        string      `json:"id" v:"required|max-length:50|regex:^[a-zA-Z0-9_-]+$"`
	Date      string      `json:"date" v:"required|datetime"`
	From      string      `json:"from" v:"required|max-length:50"`
	To        string      `json:"to" v:"required|max-length:50"`
	Amount    float64     `json:"amount" v:"required|min:0|max:1000000000000"`
	Currency  string      `json:"currency" v:"required"`
	Note      string      `json:"note" v:"max-length:500"`
	Type      string      `json:"type" v:"required|in:INCOME,EXPENSE,TRANSFER"`
	CreatedAt *gtime.Time `json:"created_at"`
	UpdatedAt *gtime.Time `json:"updated_at"`
}

type TransactionInput struct {
	Date     string  `json:"date" v:"required|datetime"`
	From     string  `json:"from" v:"required|max-length:50"`
	To       string  `json:"to" v:"required|max-length:50"`
	Amount   float64 `json:"amount" v:"required|min:0|max:1000000000000"`
	Currency string  `json:"currency" v:"required"`
	Note     string  `json:"note" v:"max-length:500"`
	Type     string  `json:"type" v:"required|in:INCOME,EXPENSE,TRANSFER"`
}

type TransactionQuery struct {
	Page      int    `json:"page" v:"min:1" d:"1"`
	Limit     int    `json:"limit" v:"min:1|max:100" d:"20"`
	StartDate string `json:"startDate" v:"date"`
	EndDate   string `json:"endDate" v:"date"`
	AccountId string `json:"accountId"`
	Type      string `json:"type" v:"in:INCOME,EXPENSE,TRANSFER"`
	SortBy    string `json:"sortBy" v:"in:date,amount,created_at" d:"date"`
	SortOrder string `json:"sortOrder" v:"in:asc,desc" d:"desc"`
}

type ListTransactionsReq struct {
	g.Meta `path:"/transactions" tags:"Transactions" method:"get" summary:"List transactions"`
	TransactionQuery
}

type ListTransactionsRes struct {
	g.Meta `mime:"application/json"`
	common.PaginatedResponse
	*common.BaseResponse
	Data []Transaction `json:"data"`
}

type CreateTransactionReq struct {
	g.Meta `path:"/transactions" tags:"Transactions" method:"post" summary:"Create a new transaction"`
	*TransactionInput
}

type CreateTransactionRes struct {
	g.Meta `mime:"application/json"`
	*Transaction
	*common.BaseResponse
}

type GetTransactionReq struct {
	g.Meta `path:"/transactions/{id}" tags:"Transactions" method:"get" summary:"Get transaction details"`
	Id     string `json:"id" v:"required"`
}

type GetTransactionRes struct {
	g.Meta `mime:"application/json"`
	*Transaction
	*common.BaseResponse
}

type UpdateTransactionReq struct {
	g.Meta `path:"/transactions/{id}" tags:"Transactions" method:"put" summary:"Update transaction"`
	Id     string `json:"id" v:"required"`
	*TransactionInput
}

type UpdateTransactionRes struct {
	g.Meta `mime:"application/json"`
	*Transaction
	*common.BaseResponse
}

type DeleteTransactionReq struct {
	g.Meta `path:"/transactions/{id}" tags:"Transactions" method:"delete" summary:"Delete transaction"`
	Id     string `json:"id" v:"required"`
}

type DeleteTransactionRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
}
