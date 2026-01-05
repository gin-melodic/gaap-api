package v1

type BaseResponse struct {
	Message string `json:"message"`
}

type PaginatedResponse struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"totalPages"`
}

type Theme struct {
	Id     string       `json:"id" v:"max-length:50|regex:^[a-z0-9-]+$"`
	Name   string       `json:"name" v:"max-length:50"`
	IsDark bool         `json:"isDark"`
	Colors *ThemeColors `json:"colors"`
}

type ThemeColors struct {
	Primary string `json:"primary" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Bg      string `json:"bg" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Card    string `json:"card" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Text    string `json:"text" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Muted   string `json:"muted" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Border  string `json:"border" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
}

type AccountTypeConfig struct {
	Label string `json:"label" v:"max-length:50"`
	Color string `json:"color" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Bg    string `json:"bg" v:"max-length:9|regex:^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"`
	Icon  string `json:"icon" v:"max-length:50"`
}

// Task is a generic task structure for API responses
type Task[P any, R any] struct {
	TaskId   string `json:"taskId"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Payload  P      `json:"payload,omitempty"`
	Result   R      `json:"result,omitempty"`
}
