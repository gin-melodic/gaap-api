// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// ExchangeRates is the golang structure of table exchange_rates for DAO operations like Where/Data.
type ExchangeRates struct {
	g.Meta        `orm:"table:exchange_rates, do:true"`
	BaseCurrency  any         //
	QuoteCurrency any         //
	Rate          any         //
	Source        any         //
	FetchedAt     *gtime.Time //
	CreatedAt     *gtime.Time //
	UpdatedAt     *gtime.Time //
}
