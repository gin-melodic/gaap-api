package model

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

type Transaction struct {
	Id            uuid.UUID
	UserId        uuid.UUID
	Date          *gtime.Time
	FromAccountId uuid.UUID
	ToAccountId   uuid.UUID
	CurrencyCode  string
	BalanceUnits  int64
	BalanceNanos  int
	Note          string
	Type          int
	CreatedAt     *gtime.Time
	UpdatedAt     *gtime.Time
}

type TransactionCreateInput struct {
	UserId        uuid.UUID `orm:"user_id"`
	Date          string    `orm:"date"`
	FromAccountId uuid.UUID `orm:"from_account_id"`
	ToAccountId   uuid.UUID `orm:"to_account_id"`
	CurrencyCode  string    `orm:"currency_code"`
	BalanceUnits  int64     `orm:"balance_units"`
	BalanceNanos  int       `orm:"balance_nanos"`
	Note          string    `orm:"note"`
	Type          int       `orm:"type"`
}

type TransactionUpdateInput struct {
	Date          string
	FromAccountId uuid.UUID
	ToAccountId   uuid.UUID
	CurrencyCode  string
	BalanceUnits  int64
	BalanceNanos  int
	Note          string
	Type          int
}

type TransactionQueryInput struct {
	Page      int
	Limit     int
	StartDate string
	EndDate   string
	AccountId uuid.UUID
	Type      int
	SortBy    string
	SortOrder string
}
