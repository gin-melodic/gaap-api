package model

import "github.com/gogf/gf/v2/os/gtime"

type Transaction struct {
	Id        string
	Date      string
	From      string
	To        string
	Amount    float64
	Currency  string
	Note      string
	Type      string
	CreatedAt *gtime.Time
	UpdatedAt *gtime.Time
}

type TransactionCreateInput struct {
	UserId   string  `orm:"user_id"`
	Date     string  `orm:"date"`
	From     string  `orm:"from_account_id"`
	To       string  `orm:"to_account_id"`
	Amount   float64 `orm:"amount"`
	Currency string  `orm:"currency"`
	Note     string  `orm:"note"`
	Type     string  `orm:"type"`
}

type TransactionUpdateInput struct {
	Date     string
	From     string
	To       string
	Amount   float64
	Currency string
	Note     string
	Type     string
}

type TransactionQueryInput struct {
	Page      int
	Limit     int
	StartDate string
	EndDate   string
	AccountId string
	Type      string
	SortBy    string
	SortOrder string
}
