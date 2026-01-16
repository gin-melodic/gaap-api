package ale

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"gaap-api/internal/crypto"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	// NonceKeyPrefix is the Redis key prefix for nonces
	NonceKeyPrefix = "ale:nonce:"
	// NonceTTL is the time-to-live for nonces (10 minutes)
	NonceTTL = 10 * time.Minute
	// TimestampTolerance is the maximum allowed time difference (5 minutes)
	TimestampTolerance = 5 * time.Minute
)

var (
	redisClient *gredis.Redis
)

// ---------------------------------------------------------
// In-memory nonce storage fallback (for development only)
// ---------------------------------------------------------

var (
	nonceStore          = make(map[string]time.Time)
	nonceStoreMu        sync.Mutex
	nonceCleanupStarted bool
)

// InitRedis initializes the Redis client for ALE operations
func InitRedis(ctx context.Context) error {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		g.Log().Warning(ctx, "REDIS_HOST not set, ALE nonce storage will use in-memory fallback")
		return nil
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	password := os.Getenv("REDIS_PASSWORD")

	address := fmt.Sprintf("%s:%s", host, port)
	g.Log().Infof(ctx, "ALE InitRedis: Connecting to Redis at %s (password set: %v)", address, password != "")

	config := &gredis.Config{
		Address: address,
		Pass:    password,
		Db:      1, // Use db 1 for ALE data (separate from locks in db 0)
	}

	redis, err := gredis.New(config)
	if err != nil {
		g.Log().Errorf(ctx, "ALE InitRedis: Failed to create Redis client: %v", err)
		return fmt.Errorf("failed to create Redis client for ALE: %w", err)
	}

	// Test connection
	if _, err := redis.Do(ctx, "PING"); err != nil {
		g.Log().Errorf(ctx, "ALE InitRedis: PING failed: %v", err)
		return fmt.Errorf("failed to connect to Redis for ALE: %w", err)
	}

	redisClient = redis
	g.Log().Info(ctx, "ALE Redis client initialized successfully (using db 1)")
	return nil
}

// CheckAndStoreNonce atomically checks if a nonce exists and stores it if not
// Returns true if nonce was successfully stored (not a replay)
// Returns false if nonce already exists (replay attack)
func CheckAndStoreNonce(ctx context.Context, nonce string) (bool, error) {
	if redisClient == nil {
		// Fallback: In-memory storage (not recommended for production)
		return checkAndStoreNonceInMemory(nonce), nil
	}

	key := NonceKeyPrefix + nonce
	ttlMs := int64(NonceTTL.Milliseconds())

	// SET key value NX PX milliseconds
	// NX: Only set if not exists
	// PX: Expire in milliseconds
	result, err := redisClient.Do(ctx, "SET", key, "1", "NX", "PX", ttlMs)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to check nonce in Redis: %v", err)
		return false, err
	}

	// SET NX returns "OK" if successful, nil if key already exists
	return result.String() == "OK", nil
}

// ValidateTimestamp checks if the timestamp is within acceptable range
func ValidateTimestamp(timestampStr string) error {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format")
	}

	requestTime := time.UnixMilli(timestamp)
	now := time.Now()
	diff := now.Sub(requestTime)

	// Check if timestamp is within tolerance (both future and past)
	if diff > TimestampTolerance || diff < -TimestampTolerance {
		return fmt.Errorf("timestamp out of range: %v difference", diff)
	}

	return nil
}

// DecryptRequest decrypts an ALE-encrypted request body
// Input format: [IV (12 bytes)][Ciphertext]
func DecryptRequest(encryptedBody []byte, hexKey string) ([]byte, error) {
	if len(encryptedBody) < crypto.NonceSize+16 {
		return nil, fmt.Errorf("encrypted body too short")
	}

	return crypto.DecryptWithHexKey(encryptedBody, hexKey)
}

// EncryptResponse encrypts response data for ALE
// Output format: [IV (12 bytes)][Ciphertext]
func EncryptResponse(plaintext []byte, hexKey string) ([]byte, error) {
	return crypto.EncryptWithHexKey(plaintext, hexKey)
}

// VerifySignature verifies the HMAC-SHA256 signature of the request
func VerifySignature(iv, ciphertext []byte, timestamp, nonce, signatureHex, hexKey string) (bool, error) {
	key, err := crypto.HexToBytes(hexKey)
	if err != nil {
		return false, err
	}

	payload := crypto.BuildSignaturePayload(iv, ciphertext, timestamp, nonce)
	return crypto.VerifyHMAC(payload, key, signatureHex), nil
}

func checkAndStoreNonceInMemory(nonce string) bool {
	nonceStoreMu.Lock()
	defer nonceStoreMu.Unlock()

	// Start cleanup goroutine if not already started
	if !nonceCleanupStarted {
		nonceCleanupStarted = true
		go cleanupExpiredNonces()
	}

	if _, exists := nonceStore[nonce]; exists {
		return false
	}

	nonceStore[nonce] = time.Now().Add(NonceTTL)
	return true
}

func cleanupExpiredNonces() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		nonceStoreMu.Lock()
		now := time.Now()
		for nonce, expiry := range nonceStore {
			if now.After(expiry) {
				delete(nonceStore, nonce)
			}
		}
		nonceStoreMu.Unlock()
	}
}
