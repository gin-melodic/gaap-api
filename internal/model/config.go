package model

type Theme struct {
	Id     string
	Name   string
	IsDark bool
	Colors ThemeColors
}

type ThemeColors struct {
	Primary string
	Bg      string
	Card    string
	Text    string
	Muted   string
	Border  string
}

type AccountTypeConfig struct {
	Label string
	Color string
	Bg    string
	Icon  string
}
