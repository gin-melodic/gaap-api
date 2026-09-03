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

		// Determine actual mode based on path
		// Public auth paths use bootstrap key, all others use session key
		actualMode := mode
		publicAuthPaths := map[string]bool{
			"/v1/auth/login":             true,
			"/v1/auth/demo-login":        true,
			"/v1/auth/register":          true,
			"/v1/auth/refresh-token":     true,
			"/v1/auth/get-currency-list": true,
		}
		if publicAuthPaths[r.URL.Path] {
			actualMode = ALEModeBootstrap
		}

		// Resolve the key before validation so protocol errors can be returned
		// using the same encrypted transport whenever the session is identifiable.
		var hexKey string
		var err error
		if actualMode == ALEModeBootstrap {
			hexKey, err = ale.GetBootstrapKey()
			if err != nil {
				g.Log().Error(ctx, "ALE bootstrap key is unavailable")
				writeProtoError(r, 500, "secure transport unavailable", "")
				return
			}
		} else {
			userId, sessionId := getSessionIdentityFromAuthHeader(r)
			if userId == "" || sessionId == "" {
				writeSessionExpiredError(r, "secure session required")
				return
			}
			hexKey, err = ale.GetSessionKey(ctx, userId, sessionId)
			if err != nil {
				writeSessionExpiredError(r, "secure session expired, please login again")
				return
			}
		}

		contentType := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
		if contentType != "application/octet-stream" {
			writeProtoError(r, 415, "ALE protobuf content type required", hexKey)
			return
		}

		// Read ALE headers
		signature := r.Header.Get(HeaderSignature)
		timestamp := r.Header.Get(HeaderTimestamp)
		nonce := r.Header.Get(HeaderNonce)

		if signature == "" || timestamp == "" || nonce == "" {
			writeProtoError(r, 400, "missing ALE headers", hexKey)
			return
		}

		// Validate timestamp
		if err := ale.ValidateTimestamp(timestamp); err != nil {
			g.Log().Warningf(ctx, "ALE timestamp validation failed: %v", err)
			writeProtoError(r, 403, "request timestamp out of range", hexKey)
			return
		}

		// Read encrypted body
		encryptedBody := r.GetBody()
		if len(encryptedBody) < crypto.NonceSize+16 {
			writeProtoError(r, 400, "invalid ALE request body", hexKey)
			return
		}

		// Extract IV and ciphertext
		iv := encryptedBody[:crypto.NonceSize]
		ciphertext := encryptedBody[crypto.NonceSize:]

		// Verify signature
		valid, err := ale.VerifySignature(iv, ciphertext, timestamp, nonce, signature, hexKey)
		if err != nil || !valid {
			writeProtoError(r, 403, "invalid ALE signature", hexKey)
			return
		}

		// Store the nonce only after authentication, preventing unauthenticated
		// traffic from exhausting the replay cache.
		isNew, err := ale.CheckAndStoreNonce(ctx, nonce)
		if err != nil {
			g.Log().Error(ctx, "ALE replay store unavailable")
			writeProtoError(r, 503, "secure transport unavailable", hexKey)
			return
		}
		if !isNew {
			writeProtoError(r, 403, "ALE request replay detected", hexKey)
			return
		}

		// Decrypt body
		plaintext, err := ale.DecryptRequest(encryptedBody, hexKey)
		if err != nil {
			writeProtoError(r, 400, "failed to decrypt ALE request", hexKey)
			return
		}

		// Store decrypted protobuf bytes in context for controller parsing
		// Controllers should use utility/proto.ParseFromALE() to extract the message
		r.SetCtxVar("ale_proto_body", plaintext)

		// Also replace request body for any fallback parsing needs
		r.Request.Body = io.NopCloser(bytes.NewReader(plaintext))
		r.Request.ContentLength = int64(len(plaintext))

		// Store the hex key in context for response encryption
		r.SetCtxVar("ale_key", hexKey)
		r.SetCtxVar("ale_enabled", true)

		// Continue to next handler
		r.Middleware.Next()

		// Note: Response encryption is handled by a separate response hook if needed
	}
}

// getUserIdFromAuthHeader extracts userId from JWT in Authorization header
func getSessionIdentityFromAuthHeader(r *ghttp.Request) (string, string) {
	authHeader := r.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		return "", ""
	}

	// Extract token (remove "Bearer " prefix)
	if authHeader[:7] != "Bearer " {
		return "", ""
	}
	tokenString := authHeader[7:]

	// Parse JWT claims without full validation (just to get userId)
	// We'll do full validation in AuthMiddleware
	claims := parseJWTClaimsUnsafe(tokenString)
	if claims == nil {
		return "", ""
	}

	userId, _ := claims["userId"].(string)
	sessionId, _ := claims["sid"].(string)
	return userId, sessionId
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
