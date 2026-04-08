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
	// Skip auth for public auth routes (login, register, refresh-token, logout)
	publicPaths := map[string]bool{
		"/v1/auth/login":         true,
		"/v1/auth/register":            true,
		"/v1/auth/refresh-token":       true,
		"/v1/auth/logout":              true,
		"/v1/auth/get-currency-list":   true,
	}
	if publicPaths[r.URL.Path] {
		r.Middleware.Next()
		return
	}

	// Get Authorization header
	authHeader := r.GetHeader("Authorization")
	if authHeader == "" {
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Authorization header required",
		})
		return
	}

	// Check Bearer prefix
	if !strings.HasPrefix(authHeader, "Bearer ") {
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Invalid authorization format, expected Bearer token",
		})
		return
	}

	// Extract token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Check if token is blacklisted
	if service.Auth().IsTokenBlacklisted(r.Context(), tokenString) {
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Token has been revoked",
		})
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
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Invalid or expired token",
		})
		return
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Invalid token claims",
		})
		return
	}

	// Check token type (must be access token)
	tokenType, _ := claims["type"].(string)
	if tokenType != "" && tokenType != "access" {
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Invalid token type, access token required",
		})
		return
	}

	// Get userId from claims
	userId, ok := claims["userId"].(string)
	if !ok || userId == "" {
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "Invalid token: missing userId",
		})
		return
	}

	// Inject userId into context
	ctx := context.WithValue(r.Context(), UserIdKey, userId)
	r.SetCtx(ctx)

	// Continue to next handler
	r.Middleware.Next()
}
