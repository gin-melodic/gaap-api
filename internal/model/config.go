package model

import "github.com/google/uuid"

type Theme struct {
	Id     uuid.UUID
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
