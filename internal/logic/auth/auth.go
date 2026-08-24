package auth

import (
	"context"
	"encoding/json"
	"net/mail"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"gaap-api/internal/ale"
	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

// getAccessTokenExpiry returns the access token expiration time from environment variables or configuration
func getAccessTokenExpiry(ctx context.Context) time.Duration {
	// Try env first
	if val := os.Getenv("JWT_ACCESS_TOKEN_EXPIRY"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	// Try config
	if v, err := g.Cfg().Get(ctx, "jwt.accessTokenExpiry"); err == nil && !v.IsEmpty() {
		if d, err := time.ParseDuration(v.String()); err == nil {
			return d
		}
	}
	return 15 * time.Minute
}

// getRefreshTokenExpiry returns the refresh token expiration time from environment variables or configuration
func getRefreshTokenExpiry(ctx context.Context) time.Duration {
	// Try env first
	if val := os.Getenv("JWT_REFRESH_TOKEN_EXPIRY"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	// Try config
	if v, err := g.Cfg().Get(ctx, "jwt.refreshTokenExpiry"); err == nil && !v.IsEmpty() {
		if d, err := time.ParseDuration(v.String()); err == nil {
			return d
		}
	}
	return 7 * 24 * time.Hour
}

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
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email == "" || in.Password == "" {
		return nil, gerror.New("email and password are required")
	}
	if in.CfTurnstileResponse == "" {
		if isProductionEnvironment() {
			return nil, gerror.New("invalid email or password")
		}
	} else if !verifyTurnstile(ctx, in.CfTurnstileResponse) {
		return nil, gerror.New("invalid email or password")
	}

	var user *entity.Users
	err = dao.Users.Ctx(ctx).
		Where(dao.Users.Columns().Email, in.Email).
		WhereNull(dao.Users.Columns().DeletedAt).
		Scan(&user)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, gerror.New("invalid email or password")
	}

	// The password arrives inside ALE and is hashed only on the server.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		return nil, gerror.New("invalid email or password")
	}

	// Verify 2FA if enabled
	if user.TwoFactorEnabled {
		if in.Code == "" {
			return nil, gerror.New("2FA code required")
		}
		valid := totp.Validate(in.Code, user.TwoFactorSecret)
		if !valid {
			return nil, gerror.New("invalid 2FA code")
		}
	}

	// Generate Token Pair
	sessionId := uuid.NewString()
	accessToken, refreshToken, err := generateTokenPair(ctx, user.Id.String(), sessionId)
	if err != nil {
		return nil, err
	}

	// Generate ALE session key
	sessionKey, err := ale.GenerateAndStoreSessionKey(ctx, user.Id.String(), sessionId)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to establish secure session")
	}

	out = &model.AuthResponse{
		Token:        accessToken, // Deprecated, for backward compatibility
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionKey:   sessionKey,
		User:         user,
	}
	return
}

func (s *sAuth) Register(ctx context.Context, in model.RegisterInput) (out *model.AuthResponse, err error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if err := validateRegistrationInput(&in); err != nil {
		return nil, err
	}
	if !isRegistrationEmailAllowed(in.Email) {
		return nil, gerror.New("registration unavailable")
	}

	// Verify Turnstile
	if in.CfTurnstileResponse == "" {
		if isProductionEnvironment() {
			return nil, gerror.New("registration unavailable")
		}
	} else {
		if !verifyTurnstile(ctx, in.CfTurnstileResponse) {
			return nil, gerror.New("invalid turnstile token")
		}
	}

	// Check email
	count, err := dao.Users.Ctx(ctx).Where(dao.Users.Columns().Email, in.Email).Count()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to check email")
	}
	if count > 0 {
		return nil, gerror.New("registration unavailable")
	}

	// Hash the ALE-protected password on the server.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to hash password")
	}

	mainCurrency := strings.ToUpper(strings.TrimSpace(in.MainCurrency))
	if mainCurrency == "" {
		mainCurrency = "USD"
	}

	currencyColumns := dao.Currencies.Columns()
	currencyCount, err := dao.Currencies.Ctx(ctx).
		Where(currencyColumns.Code, mainCurrency).
		WhereNull(currencyColumns.DeletedAt).
		Count()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to validate main currency")
	}
	if currencyCount == 0 {
		return nil, gerror.New("invalid main currency")
	}

	// Create user
	user := &entity.Users{
		Id:           uuid.New(),
		Email:        in.Email,
		Password:     string(hashedPassword),
		Nickname:     in.Nickname,
		Plan:         utils.UserLevelFree,
		MainCurrency: mainCurrency,
	}
	// Try to get a default theme
	var theme *entity.Themes
	if err := dao.Themes.Ctx(ctx).Limit(1).Scan(&theme); err == nil && theme != nil {
		user.ThemeId = theme.Id
	}

	// Insert user
	// If ThemeId is still zero (no theme found), we MUST omit it from the insert
	// so that the database receives a NULL (which is allowed) instead of a zero UUID (which violates FK)
	if user.ThemeId != uuid.Nil {
		_, err = dao.Users.Ctx(ctx).Insert(user)
	} else {
		// Construct map to exclude theme_id (implicit NULL) or set explicit NULL
		c := dao.Users.Columns()
		data := g.Map{
			c.Id:               user.Id,
			c.Email:            user.Email,
			c.Password:         user.Password,
			c.MainCurrency:     user.MainCurrency,
			c.Plan:             user.Plan,
			c.TwoFactorEnabled: user.TwoFactorEnabled,
			c.Nickname:         user.Nickname,
			c.Avatar:           user.Avatar,
			c.ThemeId:          nil,
		}
		_, err = dao.Users.Ctx(ctx).Data(data).Insert()
	}
	if err != nil {
		return nil, gerror.Wrap(err, "failed to create user")
	}

	// Generate Token Pair
	sessionId := uuid.NewString()
	accessToken, refreshToken, err := generateTokenPair(ctx, user.Id.String(), sessionId)
	if err != nil {
		return nil, err
	}

	// Generate ALE session key
	sessionKey, err := ale.GenerateAndStoreSessionKey(ctx, user.Id.String(), sessionId)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to establish secure session")
	}

	out = &model.AuthResponse{
		Token:        accessToken, // Deprecated, for backward compatibility
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionKey:   sessionKey,
		User:         user,
	}
	return
}

func validateRegistrationInput(in *model.RegisterInput) error {
	if len(in.Email) > 255 {
		return gerror.New("email must not exceed 255 characters")
	}
	if !strings.Contains(in.Email, "@") {
		return gerror.New("invalid email address")
	}
	parsed, err := mail.ParseAddress(in.Email)
	if err != nil || !strings.EqualFold(parsed.Address, in.Email) {
		return gerror.New("invalid email address")
	}
	passwordLength := utf8.RuneCountInString(in.Password)
	if passwordLength < 8 || passwordLength > 100 {
		return gerror.New("password must contain between 8 and 100 characters")
	}
	in.Nickname = strings.TrimSpace(in.Nickname)
	nicknameLength := utf8.RuneCountInString(in.Nickname)
	if nicknameLength == 0 || nicknameLength > 50 {
		return gerror.New("nickname must contain between 1 and 50 characters")
	}
	return nil
}

func (s *sAuth) Generate2FA(ctx context.Context) (out *model.TwoFactorSecret, err error) {
	userId := utils.RequireUserId(ctx)

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Scan(&user)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, gerror.New("user not found")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GAAP",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	// Save secret but don't enable yet
	_, err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Data(g.Map{
		dao.Users.Columns().TwoFactorSecret:  key.Secret(),
		dao.Users.Columns().TwoFactorEnabled: false,
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
	userId := utils.RequireUserId(ctx)

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Scan(&user)
	if err != nil {
		return err
	}

	if user.TwoFactorSecret == "" {
		return gerror.New("please generate 2FA secret first")
	}

	valid := totp.Validate(code, user.TwoFactorSecret)
	if !valid {
		return gerror.New("invalid 2FA code")
	}

	_, err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Data(g.Map{
		dao.Users.Columns().TwoFactorEnabled: true,
	}).Update()
	return
}

func (s *sAuth) Disable2FA(ctx context.Context, code string, password string) (err error) {
	userId := utils.RequireUserId(ctx)

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Scan(&user)
	if err != nil {
		return err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return gerror.New("invalid password")
	}

	// Verify code
	valid := totp.Validate(code, user.TwoFactorSecret)
	if !valid {
		return gerror.New("invalid 2FA code")
	}

	_, err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Data(g.Map{
		dao.Users.Columns().TwoFactorEnabled: false,
		dao.Users.Columns().TwoFactorSecret:  nil,
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
		return nil, gerror.New("invalid or expired refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, gerror.New("invalid token claims")
	}

	// Verify it's a refresh token
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, gerror.New("invalid token type, refresh token required")
	}

	// Check if token is blacklisted
	if IsBlacklisted(refreshTokenStr) {
		return nil, gerror.New("token has been revoked")
	}

	userId, ok := claims["userId"].(string)
	if !ok || userId == "" {
		return nil, gerror.New("invalid token: missing userId")
	}
	sessionId, ok := claims["sid"].(string)
	if !ok || sessionId == "" {
		return nil, gerror.New("invalid token: missing session")
	}

	sessionKey, err := ale.GetSessionKey(ctx, userId, sessionId)
	if err != nil {
		return nil, gerror.New("secure session expired, please login again")
	}

	accessToken, newRefreshToken, err := generateTokenPair(ctx, userId, sessionId)
	if err != nil {
		return nil, err
	}

	// Blacklist the old refresh token (token rotation)
	exp, _ := claims["exp"].(float64)
	AddToBlacklist(refreshTokenStr, time.Unix(int64(exp), 0))

	// Refresh ALE session key TTL
	if err := ale.RefreshSessionKeyTTL(ctx, userId, sessionId); err != nil {
		return nil, gerror.Wrap(err, "failed to refresh secure session")
	}

	out = &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		SessionKey:   sessionKey,
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
		expTime = time.Now().Add(getRefreshTokenExpiry(ctx))
	}

	AddToBlacklist(tokenStr, expTime)
}

// IsTokenBlacklisted checks if a token is in the blacklist
func (s *sAuth) IsTokenBlacklisted(ctx context.Context, token string) bool {
	return IsBlacklisted(token)
}

// generateTokenPair generates an access token and refresh token for a user
func generateTokenPair(ctx context.Context, userId, sessionId string) (accessToken, refreshToken string, err error) {
	// Access Token (short-lived)
	accessClaims := jwt.MapClaims{
		"userId": userId,
		"sid":    sessionId,
		"type":   "access",
		"exp":    time.Now().Add(getAccessTokenExpiry(ctx)).Unix(),
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(getJwtSecret(ctx))
	if err != nil {
		return "", "", err
	}

	// Refresh Token (long-lived)
	refreshClaims := jwt.MapClaims{
		"userId": userId,
		"sid":    sessionId,
		"type":   "refresh",
		"exp":    time.Now().Add(getRefreshTokenExpiry(ctx)).Unix(),
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(getJwtSecret(ctx))
	if err != nil {
		return "", "", err
	}

	return
}

func verifyTurnstile(ctx context.Context, token string) bool {
	// Skip verification in development mode
	if !isProductionEnvironment() && os.Getenv("ENV") == "development" {
		return true
	}

	secret := os.Getenv("TURNSTILE_SECRET")
	if secret == "" {
		v, _ := g.Cfg().Get(ctx, "turnstile.secret")
		secret = v.String()
	}

	if secret == "" {
		g.Log().Error(ctx, "Turnstile secret not configured")
		return false
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

func isProductionEnvironment() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GF_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	}
	return env == "production" || env == "prod"
}

func isRegistrationEmailAllowed(email string) bool {
	raw := strings.TrimSpace(os.Getenv("BETA_ALLOWED_EMAILS"))
	if raw == "" {
		return !isProductionEnvironment()
	}
	for _, candidate := range strings.Split(raw, ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), email) {
			return true
		}
	}
	return false
}

func (s *sAuth) UpdatePassword(ctx context.Context, password, newPassword, confirmPassword string) error {
	userId := utils.RequireUserId(ctx)

	if newPassword == "" {
		return gerror.New("new password is required")
	}

	if newPassword != confirmPassword {
		return gerror.New("new passwords do not match")
	}

	if len(newPassword) < 8 {
		return gerror.New("password must be at least 8 characters")
	}

	return g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		var user *entity.Users
		err := dbTx.Model(dao.Users.Table()).
			Where(dao.Users.Columns().Id, userId).
			Scan(&user)
		if err != nil {
			return gerror.Wrap(err, "failed to get user")
		}
		if user == nil {
			return gerror.New("user not found")
		}

		if password != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
				return gerror.New("invalid current password")
			}
		} else {
			return gerror.New("current password is required")
		}

		// newPassword is expected to be a SHA-256 hex string from frontend
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return gerror.Wrap(err, "failed to hash password")
		}

		_, err = dbTx.Model(dao.Users.Table()).
			Where(dao.Users.Columns().Id, userId).
			Data(g.Map{
				dao.Users.Columns().Password: string(hashedPassword),
			}).
			Update()
		if err != nil {
			return gerror.Wrap(err, "failed to update password")
		}

		return nil
	})
}

// GetCurrencyList returns a list of all supported currencies
func (s *sAuth) GetCurrencyList(ctx context.Context) ([]string, error) {
	var currencies []*entity.Currencies
	err := dao.Currencies.Ctx(ctx).WhereNull("deleted_at").Scan(&currencies)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get currency list")
	}

	var codes []string
	for _, c := range currencies {
		if c != nil {
			codes = append(codes, c.Code)
		}
	}
	return codes, nil
}
