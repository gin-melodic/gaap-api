// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// ExchangeRates is the golang structure for table exchange_rates.
type ExchangeRates struct {
	BaseCurrency  string      `json:"baseCurrency"  orm:"base_currency"  description:""` //
	QuoteCurrency string      `json:"quoteCurrency" orm:"quote_currency" description:""` //
	Rate          float64     `json:"rate"          orm:"rate"           description:""` //
	Source        string      `json:"source"        orm:"source"         description:""` //
	FetchedAt     *gtime.Time `json:"fetchedAt"     orm:"fetched_at"     description:""` //
	CreatedAt     *gtime.Time `json:"createdAt"     orm:"created_at"     description:""` //
	UpdatedAt     *gtime.Time `json:"updatedAt"     orm:"updated_at"     description:""` //
}
