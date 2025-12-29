package model

import "gaap-api/internal/model/entity"

type LoginInput struct {
	Email    string `json:"email" v:"required|email"`
	Password string `json:"password" v:"required|min-length:8"`
	Code     string `json:"code" v:"length:6,6"` // TOTP code
}

type RegisterInput struct {
	Email               string `json:"email" v:"required|email"`
	Password            string `json:"password" v:"required|min-length:8"`
	Nickname            string `json:"nickname" v:"required|max-length:50"`
	CfTurnstileResponse string `json:"cf_turnstile_response"`
}

type AuthResponse struct {
	Token        string        `json:"token,omitempty"`        // Deprecated: use AccessToken
	AccessToken  string        `json:"accessToken,omitempty"`  // Short-lived access token
	RefreshToken string        `json:"refreshToken,omitempty"` // Long-lived refresh token
	User         *entity.Users `json:"user"`
}

// TokenPair contains the access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type TwoFactorSecret struct {
	Secret string `json:"secret"`
	Url    string `json:"url"`
}

type Enable2FAInput struct {
	Code string `json:"code" v:"required|length:6,6"`
}

type Disable2FAInput struct {
	Code     string `json:"code" v:"required|length:6,6"`
	Password string `json:"password" v:"required"`
}
