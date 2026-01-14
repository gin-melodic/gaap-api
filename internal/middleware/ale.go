package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"

	"gaap-api/internal/ale"
	"gaap-api/internal/crypto"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ALEMode determines which key to use for encryption/decryption
type ALEMode int

const (
	// ALEModeBootstrap uses the static bootstrap key (for auth endpoints)
	ALEModeBootstrap ALEMode = iota
	// ALEModeSession uses the per-user session key (for protected endpoints)
	ALEModeSession
)

// ALE Headers
const (
	HeaderSignature = "X-Signature"
	HeaderTimestamp = "X-Timestamp"
	HeaderNonce     = "X-Nonce"
	HeaderKeyType   = "X-Key-Type" // "bootstrap" or "session"
)

// ALEMiddleware creates an ALE middleware for the specified mode
func ALEMiddleware(mode ALEMode) func(r *ghttp.Request) {
	return func(r *ghttp.Request) {
		ctx := r.Context()

		// Skip ALE for non-POST requests (GET, OPTIONS, etc.)
		if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" && r.Method != "DELETE" {
			r.Middleware.Next()
			return
		}

		// Check Content-Type - only process binary streams
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/octet-stream" {
			// Not an ALE-encrypted request, pass through
			r.Middleware.Next()
			return
		}

		// Read ALE headers
		signature := r.Header.Get(HeaderSignature)
		timestamp := r.Header.Get(HeaderTimestamp)
		nonce := r.Header.Get(HeaderNonce)

		if signature == "" || timestamp == "" || nonce == "" {
			r.Response.WriteJsonExit(g.Map{
				"code":    400,
				"message": "Missing ALE headers (X-Signature, X-Timestamp, X-Nonce)",
			})
			return
		}

		// Validate timestamp
		if err := ale.ValidateTimestamp(timestamp); err != nil {
			g.Log().Warningf(ctx, "ALE timestamp validation failed: %v", err)
			r.Response.WriteJsonExit(g.Map{
				"code":    403,
				"message": "Request timestamp out of range",
			})
			return
		}

		// Check nonce (prevent replay attacks)
		isNew, err := ale.CheckAndStoreNonce(ctx, nonce)
		if err != nil {
			g.Log().Errorf(ctx, "ALE nonce check failed: %v", err)
			r.Response.WriteJsonExit(g.Map{
				"code":    500,
				"message": "Internal server error",
			})
			return
		}
		if !isNew {
			g.Log().Warningf(ctx, "ALE replay attack detected: nonce=%s", nonce)
			r.Response.WriteJsonExit(g.Map{
				"code":    403,
				"message": "Replay attack detected: nonce already used",
			})
			return
		}

		// Read encrypted body
		encryptedBody := r.GetBody()
		if len(encryptedBody) < crypto.NonceSize+16 {
			r.Response.WriteJsonExit(g.Map{
				"code":    400,
				"message": "Request body too short for ALE decryption",
			})
			return
		}

		// Get the appropriate key
		var hexKey string
		if mode == ALEModeBootstrap {
			hexKey, err = ale.GetBootstrapKey()
			if err != nil {
				g.Log().Errorf(ctx, "Failed to get bootstrap key: %v", err)
				r.Response.WriteJsonExit(g.Map{
					"code":    500,
					"message": "ALE configuration error",
				})
				return
			}
		} else {
			// For session mode, we need the userId from JWT first
			// The JWT token is sent in Authorization header (unencrypted)
			// and only the body is encrypted

			userId := getUserIdFromAuthHeader(r)
			if userId == "" {
				r.Response.WriteJsonExit(g.Map{
					"code":    401,
					"message": "Authorization required for ALE session mode",
				})
				return
			}

			hexKey, err = ale.GetSessionKey(ctx, userId)
			if err != nil {
				g.Log().Warningf(ctx, "Failed to get session key for user %s: %v", userId, err)
				r.Response.WriteJsonExit(g.Map{
					"code":    401,
					"message": "Session key not found, please re-login",
				})
				return
			}
		}

		// Extract IV and ciphertext
		iv := encryptedBody[:crypto.NonceSize]
		ciphertext := encryptedBody[crypto.NonceSize:]

		// Verify signature
		valid, err := ale.VerifySignature(iv, ciphertext, timestamp, nonce, signature, hexKey)
		if err != nil || !valid {
			g.Log().Warningf(ctx, "ALE signature verification failed")
			r.Response.WriteJsonExit(g.Map{
				"code":    403,
				"message": "Invalid signature",
			})
			return
		}

		// Decrypt body
		plaintext, err := ale.DecryptRequest(encryptedBody, hexKey)
		if err != nil {
			g.Log().Warningf(ctx, "ALE decryption failed: %v", err)
			r.Response.WriteJsonExit(g.Map{
				"code":    400,
				"message": "Failed to decrypt request",
			})
			return
		}

		// Replace request body with decrypted content
		r.Request.Body = io.NopCloser(bytes.NewReader(plaintext))
		r.Request.ContentLength = int64(len(plaintext))
		r.Request.Header.Set("Content-Type", "application/json")

		// Store the hex key in context for response encryption
		r.SetCtxVar("ale_key", hexKey)
		r.SetCtxVar("ale_enabled", true)

		// Continue to next handler
		r.Middleware.Next()

		// Note: Response encryption is handled by a separate response hook if needed
	}
}

// getUserIdFromAuthHeader extracts userId from JWT in Authorization header
func getUserIdFromAuthHeader(r *ghttp.Request) string {
	authHeader := r.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		return ""
	}

	// Extract token (remove "Bearer " prefix)
	if authHeader[:7] != "Bearer " {
		return ""
	}
	tokenString := authHeader[7:]

	// Parse JWT claims without full validation (just to get userId)
	// We'll do full validation in AuthMiddleware
	claims := parseJWTClaimsUnsafe(tokenString)
	if claims == nil {
		return ""
	}

	userId, _ := claims["userId"].(string)
	return userId
}

// parseJWTClaimsUnsafe parses JWT claims without signature verification
// This is safe because we're only using it to get userId for key lookup
// The signature will be verified when decrypting the actual payload
func parseJWTClaimsUnsafe(tokenString string) map[string]interface{} {
	parts := splitJWT(tokenString)
	if len(parts) != 3 {
		return nil
	}

	// Decode payload (base64url)
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}

	return claims
}

func splitJWT(token string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	return base64.StdEncoding.DecodeString(s)
}
