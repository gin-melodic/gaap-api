package model

import "github.com/google/uuid"

type AccountCreateInput struct {
	UserId         uuid.UUID
	ParentId       uuid.UUID
	Name           string
	Type           int
	IsGroup        bool
	CurrencyCode   string
	Units          int64
	Nanos          int
	DefaultChildId uuid.UUID
	Date           string
	Number         string
	Remarks        string
}

type AccountUpdateInput struct {
	ParentId       uuid.UUID
	Name           string
	Type           int
	IsGroup        bool
	CurrencyCode   string
	Units          int64
	Nanos          int
	DefaultChildId uuid.UUID
	Date           string
	Number         string
	Remarks        string
}

type AccountQueryInput struct {
	Page     int
	Limit    int
	Type     int
	ParentId uuid.UUID
}
