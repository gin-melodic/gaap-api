package ale

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gaap-api/internal/crypto"
	"gaap-api/internal/redis"
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

func sessionStorageKey(userId, sessionId string) (string, error) {
	if userId == "" || sessionId == "" {
		return "", fmt.Errorf("user id and session id are required")
	}
	return SessionKeyPrefix + userId + ":" + sessionId, nil
}

func allowInMemoryFallback() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GF_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	}
	return env != "production" && env != "prod"
}

// GenerateAndStoreSessionKey generates a new session key for a user and stores it in Redis
// Returns the session key as hex string
func GenerateAndStoreSessionKey(ctx context.Context, userId, sessionId string) (string, error) {
	redisClient, _ := redis.GetRedisClient(ctx, redis.RedisTypeAle)
	sessionKey, err := crypto.GenerateSessionKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate session key: %w", err)
	}

	if redisClient == nil {
		if !allowInMemoryFallback() {
			return "", fmt.Errorf("ALE Redis is required in production")
		}
		storeSessionKeyInMemory(userId, sessionId, sessionKey)
		return sessionKey, nil
	}

	key, err := sessionStorageKey(userId, sessionId)
	if err != nil {
		return "", err
	}
	ttlMs := int64(SessionKeyTTL.Milliseconds())

	// Store session key with TTL
	_, err = redisClient.Do(ctx, "SET", key, sessionKey, "PX", ttlMs)
	if err != nil {
		return "", fmt.Errorf("failed to store session key in Redis: %w", err)
	}

	return sessionKey, nil
}

// GetSessionKey retrieves the session key for a user from Redis
func GetSessionKey(ctx context.Context, userId, sessionId string) (string, error) {
	redisClient, _ := redis.GetRedisClient(ctx, redis.RedisTypeAle)
	if redisClient == nil {
		if !allowInMemoryFallback() {
			return "", fmt.Errorf("ALE Redis is required in production")
		}
		return getSessionKeyFromMemory(userId, sessionId)
	}

	key, err := sessionStorageKey(userId, sessionId)
	if err != nil {
		return "", err
	}
	result, err := redisClient.Do(ctx, "GET", key)
	if err != nil {
		return "", fmt.Errorf("failed to get session key from Redis: %w", err)
	}

	sessionKey := result.String()
	if sessionKey == "" {
		return "", fmt.Errorf("session key not found")
	}
	return sessionKey, nil
}

// InvalidateSessionKey removes the session key for a user (on logout)
func InvalidateSessionKey(ctx context.Context, userId, sessionId string) error {
	redisClient, _ := redis.GetRedisClient(ctx, redis.RedisTypeAle)
	if redisClient == nil {
		if !allowInMemoryFallback() {
			return fmt.Errorf("ALE Redis is required in production")
		}
		removeSessionKeyFromMemory(userId, sessionId)
		return nil
	}

	key, err := sessionStorageKey(userId, sessionId)
	if err != nil {
		return err
	}
	_, err = redisClient.Do(ctx, "DEL", key)
	if err != nil {
		return fmt.Errorf("failed to invalidate session key: %w", err)
	}

	return nil
}

// RefreshSessionKeyTTL extends the TTL of a session key (called on token refresh)
func RefreshSessionKeyTTL(ctx context.Context, userId, sessionId string) error {
	redisClient, _ := redis.GetRedisClient(ctx, redis.RedisTypeAle)
	if redisClient == nil {
		if !allowInMemoryFallback() {
			return fmt.Errorf("ALE Redis is required in production")
		}
		return nil
	}

	key, err := sessionStorageKey(userId, sessionId)
	if err != nil {
		return err
	}
	ttlMs := int64(SessionKeyTTL.Milliseconds())

	// Extend TTL
	_, err = redisClient.Do(ctx, "PEXPIRE", key, ttlMs)
	if err != nil {
		return fmt.Errorf("failed to refresh session key TTL: %w", err)
	}

	return nil
}

func storeSessionKeyInMemory(userId, sessionId, sessionKey string) {
	sessionKeyStoreMu.Lock()
	defer sessionKeyStoreMu.Unlock()
	sessionKeyStore[userId+":"+sessionId] = sessionKey
}

func getSessionKeyFromMemory(userId, sessionId string) (string, error) {
	sessionKeyStoreMu.RLock()
	defer sessionKeyStoreMu.RUnlock()
	sessionKey, exists := sessionKeyStore[userId+":"+sessionId]
	if !exists {
		return "", fmt.Errorf("session key not found")
	}
	return sessionKey, nil
}

func removeSessionKeyFromMemory(userId, sessionId string) {
	sessionKeyStoreMu.Lock()
	defer sessionKeyStoreMu.Unlock()
	delete(sessionKeyStore, userId+":"+sessionId)
}
