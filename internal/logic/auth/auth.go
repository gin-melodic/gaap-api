package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type sAuth struct{}

func init() {
	service.RegisterAuth(New())
}

func New() *sAuth {
	return &sAuth{}
}

const (
	// AccessTokenExpiry is the expiration time for access tokens (15 minutes)
	AccessTokenExpiry = 15 * time.Minute
	// RefreshTokenExpiry is the expiration time for refresh tokens (7 days)
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// getJwtSecret returns the JWT secret from environment variables or configuration
func getJwtSecret(ctx context.Context) []byte {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return []byte(secret)
	}
	// Fallback to config
	v, _ := g.Cfg().Get(ctx, "jwt.secret")
	return v.Bytes()
}

func (s *sAuth) Login(ctx context.Context, in model.LoginInput) (out *model.AuthResponse, err error) {
	if in.Email == "" || in.Password == "" { // Check if email and password are provided
		return nil, errors.New("email and password are required")
	}

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where("email", in.Email).Scan(&user)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	if strings.TrimSpace(in.Password) != strings.TrimSpace(user.Password) {
		return nil, errors.New("invalid email or password")
	}

	// Verify 2FA if enabled
	if user.TwoFactorEnabled {
		if in.Code == "" {
			return nil, errors.New("2FA code required")
		}
		valid := totp.Validate(in.Code, user.TwoFactorSecret)
		if !valid {
			return nil, errors.New("invalid 2FA code")
		}
	}

	// Generate Token Pair
	accessToken, refreshToken, err := generateTokenPair(user.Id)
	if err != nil {
		return nil, err
	}

	out = &model.AuthResponse{
		Token:        accessToken, // Deprecated, for backward compatibility
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}
	return
}

func (s *sAuth) Register(ctx context.Context, in model.RegisterInput) (out *model.AuthResponse, err error) {
	// Verify Turnstile
	if in.CfTurnstileResponse == "" {
		// For development, we might skip if configured, but for now enforce it or check config
		// return nil, errors.New("turnstile token required")
	} else {
		if !verifyTurnstile(ctx, in.CfTurnstileResponse) {
			return nil, errors.New("invalid turnstile token")
		}
	}

	// Check email
	count, err := dao.Users.Ctx(ctx).Where("email", in.Email).Count()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	// Use g.Map to avoid sending empty ID, letting DB generate it
	_, err = dao.Users.Ctx(ctx).Data(g.Map{
		"email":    in.Email,
		"nickname": in.Nickname,
		"plan":     "FREE",
		"password": string(hashedPassword),
	}).Insert()
	if err != nil {
		return nil, err
	}

	// Fetch the created user to get the generated ID
	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where("email", in.Email).Scan(&user)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("failed to create user")
	}

	// Generate Token Pair
	accessToken, refreshToken, err := generateTokenPair(user.Id)
	if err != nil {
		return nil, err
	}

	out = &model.AuthResponse{
		Token:        accessToken, // Deprecated, for backward compatibility
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}
	return
}

func (s *sAuth) Generate2FA(ctx context.Context) (out *model.TwoFactorSecret, err error) {
	// Get current user ID from context (assuming middleware sets it)
	userId := ctx.Value("userId")
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where("id", userId).Scan(&user)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GAAP",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	// Save secret but don't enable yet
	_, err = dao.Users.Ctx(ctx).Where("id", userId).Data(g.Map{
		"two_factor_secret":  key.Secret(),
		"two_factor_enabled": false,
	}).Update()
	if err != nil {
		return nil, err
	}

	out = &model.TwoFactorSecret{
		Secret: key.Secret(),
		Url:    key.URL(),
	}
	return
}

func (s *sAuth) Enable2FA(ctx context.Context, code string) (err error) {
	userId := ctx.Value("userId")
	if userId == nil {
		return errors.New("unauthorized")
	}

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where("id", userId).Scan(&user)
	if err != nil {
		return err
	}

	if user.TwoFactorSecret == "" {
		return errors.New("please generate 2FA secret first")
	}

	valid := totp.Validate(code, user.TwoFactorSecret)
	if !valid {
		return errors.New("invalid 2FA code")
	}

	_, err = dao.Users.Ctx(ctx).Where("id", userId).Data(g.Map{
		"two_factor_enabled": true,
	}).Update()
	return
}

func (s *sAuth) Disable2FA(ctx context.Context, code string, password string) (err error) {
	userId := ctx.Value("userId")
	if userId == nil {
		return errors.New("unauthorized")
	}

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where("id", userId).Scan(&user)
	if err != nil {
		return err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return errors.New("invalid password")
	}

	// Verify code
	valid := totp.Validate(code, user.TwoFactorSecret)
	if !valid {
		return errors.New("invalid 2FA code")
	}

	_, err = dao.Users.Ctx(ctx).Where("id", userId).Data(g.Map{
		"two_factor_enabled": false,
		"two_factor_secret":  nil,
	}).Update()
	return
}

// RefreshToken validates a refresh token and returns a new token pair
func (s *sAuth) RefreshToken(ctx context.Context, refreshTokenStr string) (out *model.TokenPair, err error) {
	// Parse and validate the refresh token
	token, err := jwt.Parse(refreshTokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return getJwtSecret(ctx), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Verify it's a refresh token
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, errors.New("invalid token type, refresh token required")
	}

	// Check if token is blacklisted
	if IsBlacklisted(refreshTokenStr) {
		return nil, errors.New("token has been revoked")
	}

	userId, ok := claims["userId"].(string)
	if !ok || userId == "" {
		return nil, errors.New("invalid token: missing userId")
	}

	// Generate new token pair
	accessToken, newRefreshToken, err := generateTokenPair(userId)
	if err != nil {
		return nil, err
	}

	// Blacklist the old refresh token (token rotation)
	exp, _ := claims["exp"].(float64)
	AddToBlacklist(refreshTokenStr, time.Unix(int64(exp), 0))

	out = &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}
	return
}

// AddTokenToBlacklist adds a token to the blacklist
func (s *sAuth) AddTokenToBlacklist(ctx context.Context, tokenStr string) {
	// Strip Bearer prefix if present
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

	// Parse token to get expiration
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return getJwtSecret(context.Background()), nil
	})

	var expTime time.Time
	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if exp, ok := claims["exp"].(float64); ok {
				expTime = time.Unix(int64(exp), 0)
			}
		}
	}

	// Default expiration if we couldn't parse it
	if expTime.IsZero() {
		expTime = time.Now().Add(RefreshTokenExpiry)
	}

	AddToBlacklist(tokenStr, expTime)
}

// IsTokenBlacklisted checks if a token is in the blacklist
func (s *sAuth) IsTokenBlacklisted(ctx context.Context, token string) bool {
	return IsBlacklisted(token)
}

// generateTokenPair generates an access token and refresh token for a user
func generateTokenPair(userId string) (accessToken, refreshToken string, err error) {
	// Access Token (short-lived)
	accessClaims := jwt.MapClaims{
		"userId": userId,
		"type":   "access",
		"exp":    time.Now().Add(AccessTokenExpiry).Unix(),
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(getJwtSecret(context.Background()))
	if err != nil {
		return "", "", err
	}

	// Refresh Token (long-lived)
	refreshClaims := jwt.MapClaims{
		"userId": userId,
		"type":   "refresh",
		"exp":    time.Now().Add(RefreshTokenExpiry).Unix(),
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(getJwtSecret(context.Background()))
	if err != nil {
		return "", "", err
	}

	return
}

func verifyTurnstile(ctx context.Context, token string) bool {
	// Skip verification in development mode
	if os.Getenv("ENV") == "development" {
		return true
	}

	secret := os.Getenv("TURNSTILE_SECRET")
	if secret == "" {
		v, _ := g.Cfg().Get(ctx, "turnstile.secret")
		secret = v.String()
	}

	if secret == "" {
		g.Log().Warning(ctx, "Turnstile secret not configured (env or config), skipping verification")
		return true
	}

	// Call Cloudflare Turnstile API
	resp, err := g.Client().Post(ctx,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		g.Map{"secret": secret, "response": token})
	if err != nil {
		g.Log().Errorf(ctx, "Turnstile verification failed: %v", err)
		return false
	}
	defer resp.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(resp.ReadAll(), &result); err != nil {
		g.Log().Errorf(ctx, "Failed to parse Turnstile response: %v", err)
		return false
	}

	success, _ := result["success"].(bool)
	return success
}
