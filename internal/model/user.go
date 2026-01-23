package model

type UserProfile struct {
	Email            string
	Nickname         string
	Avatar           string
	Plan             int
	TwoFactorEnabled bool
	MainCurrency     string
}

type UserUpdateInput struct {
	Nickname     string `json:"nickname"`
	Avatar       string `json:"avatar"`
	Plan         int    `json:"plan"`
	MainCurrency string `json:"mainCurrency"`
}
