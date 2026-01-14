package ale

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"gaap-api/internal/crypto"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	// SessionKeyPrefix is the Redis key prefix for session keys
	SessionKeyPrefix = "ale:session:"
	// SessionKeyTTL is the time-to-live for session keys (7 days, matching refresh token)
	SessionKeyTTL = 7 * 24 * time.Hour
)

// ---------------------------------------------------------
// In-memory session key storage fallback (for development only)
// ---------------------------------------------------------

var (
	sessionKeyStore   = make(map[string]string)
	sessionKeyStoreMu sync.RWMutex
)

// GetBootstrapKey returns the bootstrap key for auth endpoints
// This is a static key used only for login/register/refresh endpoints
func GetBootstrapKey() (string, error) {
	key := os.Getenv("ALE_BOOTSTRAP_KEY")
	if key == "" {
		return "", fmt.Errorf("ALE_BOOTSTRAP_KEY environment variable not set")
	}

	// Validate key length (should be 64 hex chars = 32 bytes = 256 bits)
	if len(key) != 64 {
		return "", fmt.Errorf("ALE_BOOTSTRAP_KEY must be 64 hex characters (256-bit key)")
	}

	// Validate it's valid hex
	if _, err := crypto.HexToBytes(key); err != nil {
		return "", fmt.Errorf("ALE_BOOTSTRAP_KEY must be valid hex: %w", err)
	}

	return key, nil
}

// GenerateAndStoreSessionKey generates a new session key for a user and stores it in Redis
// Returns the session key as hex string
func GenerateAndStoreSessionKey(ctx context.Context, userId string) (string, error) {
	sessionKey, err := crypto.GenerateSessionKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate session key: %w", err)
	}

	if redisClient == nil {
		// Fallback: store in memory (for development)
		storeSessionKeyInMemory(userId, sessionKey)
		g.Log().Debugf(ctx, "Stored session key in memory for user %s (Redis not available)", userId)
		return sessionKey, nil
	}

	key := SessionKeyPrefix + userId
	ttlMs := int64(SessionKeyTTL.Milliseconds())

	// Store session key with TTL
	_, err = redisClient.Do(ctx, "SET", key, sessionKey, "PX", ttlMs)
	if err != nil {
		return "", fmt.Errorf("failed to store session key in Redis: %w", err)
	}

	g.Log().Debugf(ctx, "Stored session key in Redis for user %s", userId)
	return sessionKey, nil
}

// GetSessionKey retrieves the session key for a user from Redis
func GetSessionKey(ctx context.Context, userId string) (string, error) {
	if redisClient == nil {
		// Fallback: get from memory
		return getSessionKeyFromMemory(userId)
	}

	key := SessionKeyPrefix + userId
	result, err := redisClient.Do(ctx, "GET", key)
	if err != nil {
		return "", fmt.Errorf("failed to get session key from Redis: %w", err)
	}

	sessionKey := result.String()
	if sessionKey == "" {
		return "", fmt.Errorf("session key not found for user %s", userId)
	}

	return sessionKey, nil
}

// InvalidateSessionKey removes the session key for a user (on logout)
func InvalidateSessionKey(ctx context.Context, userId string) error {
	if redisClient == nil {
		// Fallback: remove from memory
		removeSessionKeyFromMemory(userId)
		return nil
	}

	key := SessionKeyPrefix + userId
	_, err := redisClient.Do(ctx, "DEL", key)
	if err != nil {
		return fmt.Errorf("failed to invalidate session key: %w", err)
	}

	g.Log().Debugf(ctx, "Invalidated session key for user %s", userId)
	return nil
}

// RefreshSessionKeyTTL extends the TTL of a session key (called on token refresh)
func RefreshSessionKeyTTL(ctx context.Context, userId string) error {
	if redisClient == nil {
		// Memory storage doesn't have TTL, skip
		return nil
	}

	key := SessionKeyPrefix + userId
	ttlMs := int64(SessionKeyTTL.Milliseconds())

	// Extend TTL
	_, err := redisClient.Do(ctx, "PEXPIRE", key, ttlMs)
	if err != nil {
		return fmt.Errorf("failed to refresh session key TTL: %w", err)
	}

	return nil
}

func storeSessionKeyInMemory(userId, sessionKey string) {
	sessionKeyStoreMu.Lock()
	defer sessionKeyStoreMu.Unlock()
	sessionKeyStore[userId] = sessionKey
}

func getSessionKeyFromMemory(userId string) (string, error) {
	sessionKeyStoreMu.RLock()
	defer sessionKeyStoreMu.RUnlock()
	sessionKey, exists := sessionKeyStore[userId]
	if !exists {
		return "", fmt.Errorf("session key not found for user %s", userId)
	}
	return sessionKey, nil
}

func removeSessionKeyFromMemory(userId string) {
	sessionKeyStoreMu.Lock()
	defer sessionKeyStoreMu.Unlock()
	delete(sessionKeyStore, userId)
}
