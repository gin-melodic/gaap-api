package model

type UserProfile struct {
	Email            string
	Nickname         string
	Avatar           string
	Plan             string
	TwoFactorEnabled bool
	MainCurrency     string
}

type UserUpdateInput struct {
	Nickname string
	Avatar   string
	Plan     string
}
