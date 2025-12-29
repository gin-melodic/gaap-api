package auth

import (
	"sync"
	"time"
)

// tokenBlacklist stores revoked tokens with their expiration times
var tokenBlacklist = struct {
	sync.RWMutex
	tokens map[string]time.Time
}{
	tokens: make(map[string]time.Time),
}

// cleanupInterval defines how often to clean expired tokens
const cleanupInterval = 5 * time.Minute

func init() {
	// Start background cleanup goroutine
	go cleanupExpiredTokens()
}

// cleanupExpiredTokens removes expired tokens from the blacklist periodically
func cleanupExpiredTokens() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		tokenBlacklist.Lock()
		for token, expAt := range tokenBlacklist.tokens {
			if now.After(expAt) {
				delete(tokenBlacklist.tokens, token)
			}
		}
		tokenBlacklist.Unlock()
	}
}

// AddToBlacklist adds a token to the blacklist with its expiration time
func AddToBlacklist(token string, expAt time.Time) {
	tokenBlacklist.Lock()
	defer tokenBlacklist.Unlock()
	tokenBlacklist.tokens[token] = expAt
}

// IsBlacklisted checks if a token is in the blacklist
func IsBlacklisted(token string) bool {
	tokenBlacklist.RLock()
	defer tokenBlacklist.RUnlock()

	expAt, exists := tokenBlacklist.tokens[token]
	if !exists {
		return false
	}

	// If token has expired, it's effectively not blacklisted anymore
	// (it would be invalid anyway)
	if time.Now().After(expAt) {
		return false
	}

	return true
}
