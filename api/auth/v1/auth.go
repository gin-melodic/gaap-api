package v1

import (
	common "gaap-api/api/common/v1"
	v1 "gaap-api/api/user/v1"

	"github.com/gogf/gf/v2/frame/g"
)

type AuthResponse struct {
	Token        string   `json:"token,omitempty"`        // Deprecated: use AccessToken
	AccessToken  string   `json:"accessToken,omitempty"`  // Short-lived access token
	RefreshToken string   `json:"refreshToken,omitempty"` // Long-lived refresh token
	User         *v1.User `json:"user"`
}

type LoginReq struct {
	g.Meta              `path:"/v1/auth/login" tags:"Authentication" method:"post" summary:"User login"`
	Email               string `json:"email" v:"required|email"`
	Password            string `json:"password" v:"required|min-length:8"`
	Code                string `json:"code" v:"length:6,6"`
	CfTurnstileResponse string `json:"cf_turnstile_response" v:"required"`
}

type LoginRes struct {
	g.Meta `mime:"application/json"`
	*AuthResponse
	*common.BaseResponse
}

type RegisterReq struct {
	g.Meta              `path:"/v1/auth/register" tags:"Authentication" method:"post" summary:"User registration"`
	Email               string `json:"email" v:"required|email"`
	Password            string `json:"password" v:"required|min-length:8"`
	Nickname            string `json:"nickname" v:"required|max-length:50"`
	CfTurnstileResponse string `json:"cf_turnstile_response"`
}

type RegisterRes struct {
	g.Meta `mime:"application/json"`
	*AuthResponse
	*common.BaseResponse
}

type LogoutReq struct {
	g.Meta `path:"/v1/auth/logout" tags:"Authentication" method:"post" summary:"User logout"`
}

type LogoutRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
}

type RefreshTokenReq struct {
	g.Meta       `path:"/v1/auth/refresh" tags:"Authentication" method:"post" summary:"Refresh access token"`
	RefreshToken string `json:"refreshToken" v:"required"`
}

type RefreshTokenRes struct {
	g.Meta       `mime:"application/json"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	*common.BaseResponse
}

type TwoFactorSecret struct {
	Secret string `json:"secret"`
	Url    string `json:"url"`
}

type Generate2FAReq struct {
	g.Meta `path:"/v1/auth/2fa/generate" tags:"Authentication" method:"post" summary:"Generate 2FA secret"`
}

type Generate2FARes struct {
	g.Meta `mime:"application/json"`
	*TwoFactorSecret
	*common.BaseResponse
}

type Enable2FAReq struct {
	g.Meta `path:"/v1/auth/2fa/enable" tags:"Authentication" method:"post" summary:"Enable 2FA"`
	Code   string `json:"code" v:"required|length:6,6"`
}

type Enable2FARes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
}

type Disable2FAReq struct {
	g.Meta   `path:"/v1/auth/2fa/disable" tags:"Authentication" method:"post" summary:"Disable 2FA"`
	Code     string `json:"code" v:"required|length:6,6"`
	Password string `json:"password" v:"required"`
}

type Disable2FARes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
}
