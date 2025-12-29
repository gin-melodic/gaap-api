package model

type DashboardSummary struct {
	Assets      float64
	Liabilities float64
	NetWorth    float64
}

type MonthlyStats struct {
	Income  float64
	Expense float64
}

type DailyBalance struct {
	Date     string
	Balances map[string]float64
}
