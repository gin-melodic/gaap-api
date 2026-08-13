package middleware

import (
	"context"
	"os"
	"strings"

	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserIdKey is the context key for storing user ID
	UserIdKey ContextKey = "userId"
	// SessionIdKey identifies the browser/device ALE session.
	SessionIdKey ContextKey = "sessionId"
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

// AuthMiddleware validates JWT token and injects userId into context
func AuthMiddleware(r *ghttp.Request) {
	reject := func(status int, message string) {
		writeProtoError(r, status, message, r.GetCtxVar("ale_key").String())
	}
	// Skip auth for public auth routes (login, register, refresh-token, logout)
	publicPaths := map[string]bool{
		"/v1/auth/login":         true,
		"/v1/auth/register":            true,
		"/v1/auth/refresh-token":       true,
		"/v1/auth/get-currency-list":   true,
	}
	if publicPaths[r.URL.Path] {
		r.Middleware.Next()
		return
	}

	// Get Authorization header
	authHeader := r.GetHeader("Authorization")
	if authHeader == "" {
		reject(401, "authorization required")
		return
	}

	// Check Bearer prefix
	if !strings.HasPrefix(authHeader, "Bearer ") {
		reject(401, "invalid authorization format")
		return
	}

	// Extract token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Check if token is blacklisted
	if service.Auth().IsTokenBlacklisted(r.Context(), tokenString) {
		reject(401, "token has been revoked")
		return
	}

	// Parse and validate JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return getJwtSecret(r.Context()), nil
	})

	if err != nil || !token.Valid {
		reject(401, "invalid or expired token")
		return
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		reject(401, "invalid token claims")
		return
	}

	// Check token type (must be access token)
	tokenType, _ := claims["type"].(string)
	if tokenType != "" && tokenType != "access" {
		reject(401, "access token required")
		return
	}

	// Get userId from claims
	userId, ok := claims["userId"].(string)
	if !ok || userId == "" {
		reject(401, "invalid token claims")
		return
	}

	sessionId, ok := claims["sid"].(string)
	if !ok || sessionId == "" {
		reject(401, "invalid token session")
		return
	}

	// Inject userId into context
	ctx := context.WithValue(r.Context(), UserIdKey, userId)
	ctx = context.WithValue(ctx, SessionIdKey, sessionId)
	r.SetCtx(ctx)

	// Continue to next handler
	r.Middleware.Next()
}
