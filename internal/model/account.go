package model

import "github.com/gogf/gf/v2/os/gtime"

type Account struct {
	Id             string
	ParentId       string
	Name           string
	Type           string
	IsGroup        bool
	Balance        float64
	Currency       string
	DefaultChildId string
	Date           string
	Number         string
	Remarks        string
	CreatedAt      *gtime.Time
	UpdatedAt      *gtime.Time
}

type AccountCreateInput struct {
	UserId         string
	ParentId       string
	Name           string
	Type           string
	IsGroup        bool
	Balance        float64
	Currency       string
	DefaultChildId string
	Date           string
	Number         string
	Remarks        string
}

type AccountUpdateInput struct {
	ParentId       string
	Name           string
	Type           string
	IsGroup        bool
	Balance        float64
	Currency       string
	DefaultChildId string
	Date           string
	Number         string
	Remarks        string
}

type AccountQueryInput struct {
	Page     int
	Limit    int
	Type     string
	ParentId string
}
