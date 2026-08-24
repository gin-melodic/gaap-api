package model

type DashboardSummary struct {
	AssetsUnits      int64
	AssetsNanos      int32
	LiabilitiesUnits int64
	LiabilitiesNanos int32
	NetWorthUnits    int64
	NetWorthNanos    int32
	CurrencyCode     string
}

type MonthlyStats struct {
	IncomeUnits  int64
	IncomeNanos  int32
	ExpenseUnits int64
	ExpenseNanos int32
	CurrencyCode string
}

type DailyBalance struct {
	Date     string
	Balances map[string]DailyAccountBalance
}

type DailyAccountBalance struct {
	Units        int64
	Nanos        int32
	CurrencyCode string
}
