package utils_test

import (
	"testing"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/test/gtest"
)

func Test_MoneyHelper(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// 1. From Entity
		ent := &entity.Accounts{
			BalanceUnits: 100,
			BalanceNanos: 500_000_000,
			CurrencyCode: "CNY",
		}
		mh := utils.NewFromEntity(ent)
		g.Assert(mh.Currency, "CNY")
		g.Assert(mh.String(), "100.5")

		// 2. To Entity
		u, n := mh.ToEntityValues()
		g.Assert(u, int64(100))
		g.Assert(n, int32(500_000_000))

		// 3. Add
		other := utils.NewFromEntity(&entity.Accounts{
			BalanceUnits: 50,
			BalanceNanos: 250_000_000,
			CurrencyCode: "CNY",
		})
		sum, err := mh.Add(other)
		g.AssertNil(err)
		g.Assert(sum.String(), "150.75")

		su, sn := sum.ToEntityValues()
		g.Assert(su, int64(150))
		g.Assert(sn, int32(750_000_000))

		// 4. Sub
		diff, err := mh.Sub(other)
		g.AssertNil(err)
		g.Assert(diff.String(), "50.25")

		// 5. Negative values
		negEnt := &entity.Accounts{
			BalanceUnits: -10,
			BalanceNanos: -500_000_000,
			CurrencyCode: "CNY",
		}
		negMh := utils.NewFromEntity(negEnt)
		g.Assert(negMh.String(), "-10.5")

		added, err := mh.Add(negMh) // 100.5 + (-10.5) = 90.0
		g.AssertNil(err)
		g.Assert(added.String(), "90")
	})
}
