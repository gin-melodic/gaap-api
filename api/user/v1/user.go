package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
)

type User struct {
	Email            string  `json:"email" v:"required|email|max-length:255"`
	Nickname         string  `json:"nickname" v:"required|max-length:50"`
	Avatar           *string `json:"avatar" v:"max-length:2048"`
	Plan             string  `json:"plan" v:"required|in:FREE,PRO"`
	TwoFactorEnabled bool    `json:"twoFactorEnabled"`
	MainCurrency     string  `json:"mainCurrency"`
}

type UserInput struct {
	Nickname string  `json:"nickname" v:"max-length:50"`
	Avatar   *string `json:"avatar" v:"max-length:2048"`
	Plan     string  `json:"plan" v:"in:FREE,PRO"`
}

type GetUserProfileReq struct {
	g.Meta `path:"/user/profile" tags:"User" method:"get" summary:"Get current user profile"`
}

type GetUserProfileRes struct {
	g.Meta `mime:"application/json"`
	User   *User `json:"user"`
	*common.BaseResponse
}

type UpdateUserProfileReq struct {
	g.Meta `path:"/user/profile" tags:"User" method:"put" summary:"Update user profile"`
	*UserInput
}

type UpdateUserProfileRes struct {
	g.Meta `mime:"application/json"`
	User   *User `json:"user"`
	*common.BaseResponse
}

type UpdateThemePreferenceReq struct {
	g.Meta `path:"/user/preferences/theme" tags:"User" method:"put" summary:"Update user theme preference"`
	*common.Theme
}

type UpdateThemePreferenceRes struct {
	g.Meta `mime:"application/json"`
	*common.Theme
	*common.BaseResponse
}
