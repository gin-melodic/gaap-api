package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/hkdf"
)

const (
	// KeySize is the size of AES-256 key in bytes
	KeySize = 32
	// NonceSize is the size of GCM nonce in bytes
	NonceSize = 12
	// HMACSize is the size of HMAC-SHA256 output in bytes
	HMACSize = 32
)

var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext: too short")
	ErrDecryptionFailed  = errors.New("decryption failed: authentication error")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrInvalidKeySize    = errors.New("invalid key size: must be 32 bytes")
	ErrInvalidHexKey     = errors.New("invalid hex key format")
)

// ---------------------------------------------------------
// Hex Encoding Helpers
// ---------------------------------------------------------

// HexToBytes converts a hex string to bytes
func HexToBytes(hexStr string) ([]byte, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, ErrInvalidHexKey
	}
	return bytes, nil
}

// BytesToHex converts bytes to hex string
func BytesToHex(bytes []byte) string {
	return hex.EncodeToString(bytes)
}

// ---------------------------------------------------------
// Session Key Generation
// ---------------------------------------------------------

// GenerateSessionKey generates a random 256-bit session key and returns it as hex string
func GenerateSessionKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}
	return BytesToHex(key), nil
}

// ---------------------------------------------------------
// HMAC-SHA256 Signing and Verification
// ---------------------------------------------------------

// SignHMAC signs data using HMAC-SHA256 with the provided key
// Returns signature as hex string
func SignHMAC(data, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return BytesToHex(h.Sum(nil))
}

// VerifyHMAC verifies HMAC-SHA256 signature
// signatureHex is the expected signature in hex format
func VerifyHMAC(data, key []byte, signatureHex string) bool {
	expectedSig, err := HexToBytes(signatureHex)
	if err != nil || len(expectedSig) != HMACSize {
		return false
	}

	h := hmac.New(sha256.New, key)
	h.Write(data)
	actualSig := h.Sum(nil)

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(actualSig, expectedSig) == 1
}

// BuildSignaturePayload builds the payload for HMAC signing
// Order must match frontend: IV + Ciphertext + Timestamp + Nonce
func BuildSignaturePayload(iv, ciphertext []byte, timestamp, nonce string) []byte {
	timestampBytes := []byte(timestamp)
	nonceBytes := []byte(nonce)

	payload := make([]byte, len(iv)+len(ciphertext)+len(timestampBytes)+len(nonceBytes))
	offset := 0

	copy(payload[offset:], iv)
	offset += len(iv)
	copy(payload[offset:], ciphertext)
	offset += len(ciphertext)
	copy(payload[offset:], timestampBytes)
	offset += len(timestampBytes)
	copy(payload[offset:], nonceBytes)

	return payload
}

// ---------------------------------------------------------
// AES-GCM with Hex Key Support
// ---------------------------------------------------------

// EncryptWithHexKey encrypts plaintext using AES-GCM with hex-encoded key
// Returns: IV (12 bytes) concatenated with ciphertext
func EncryptWithHexKey(plaintext []byte, hexKey string) ([]byte, error) {
	key, err := HexToBytes(hexKey)
	if err != nil {
		return nil, err
	}
	return Encrypt(plaintext, key)
}

// DecryptWithHexKey decrypts ciphertext using AES-GCM with hex-encoded key
// Expects: IV (12 bytes) concatenated with ciphertext
func DecryptWithHexKey(ciphertext []byte, hexKey string) ([]byte, error) {
	key, err := HexToBytes(hexKey)
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, key)
}

// DeriveKey derives a unique encryption key from userId and server secret using HKDF-SHA256.
// This ensures each user has a unique key, preventing cross-user data import.
func DeriveKey(userId, serverSecret string) ([]byte, error) {
	// Combine userId and serverSecret as input key material
	ikm := []byte(serverSecret + userId)
	salt := []byte("gaap-data-export-v1")
	info := []byte(userId)

	// Use HKDF to derive a secure key
	reader := hkdf.New(sha256.New, ikm, salt, info)
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GetServerSecret reads the server secret from environment variable.
// In production, this should be set securely via environment configuration.
func GetServerSecret() string {
	secret := os.Getenv("EXPORT_SECRET")
	if secret == "" {
		// Fallback to JWT_SECRET if EXPORT_SECRET is not set
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		g.Log().Fatal(context.Background(), "server secret not found")
	}
	return secret
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// Returns ciphertext in format: nonce (12 bytes) || encrypted data || auth tag (16 bytes)
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, errors.New("invalid key size: must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends ciphertext to nonce, so result is: nonce || ciphertext || tag
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided key.
// Expects ciphertext in format: nonce (12 bytes) || encrypted data || auth tag (16 bytes)
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, errors.New("invalid key size: must be 32 bytes")
	}

	// Minimum size: nonce (12) + tag (16) = 28 bytes
	if len(ciphertext) < NonceSize+16 {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:NonceSize]
	encryptedData := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptFile encrypts a file and writes to output path.
func EncryptFile(inputPath, outputPath string, key []byte) error {
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, ciphertext, 0600)
}

// DecryptFile decrypts a file and writes to output path.
func DecryptFile(inputPath, outputPath string, key []byte) error {
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	plaintext, err := Decrypt(ciphertext, key)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, plaintext, 0600)
}
