package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"
)

const (
	// KeySize is the size of AES-256 key in bytes
	KeySize = 32
	// NonceSize is the size of GCM nonce in bytes
	NonceSize = 12
)

var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext: too short")
	ErrDecryptionFailed  = errors.New("decryption failed: authentication error")
)

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
		// Default for development only - should never be used in production
		secret = "gaap-dev-secret-change-in-production"
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
