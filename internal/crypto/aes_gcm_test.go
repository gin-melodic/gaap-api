package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name         string
		userId       string
		serverSecret string
		wantLen      int
	}{
		{
			name:         "normal user",
			userId:       "user-001",
			serverSecret: "test-secret",
			wantLen:      KeySize,
		},
		{
			name:         "empty userId",
			userId:       "",
			serverSecret: "test-secret",
			wantLen:      KeySize,
		},
		{
			name:         "long userId",
			userId:       "user-with-very-long-id-123456789012345678901234567890",
			serverSecret: "test-secret",
			wantLen:      KeySize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := DeriveKey(tt.userId, tt.serverSecret)
			if err != nil {
				t.Fatalf("DeriveKey() error = %v", err)
			}
			if len(key) != tt.wantLen {
				t.Errorf("DeriveKey() key length = %d, want %d", len(key), tt.wantLen)
			}
		})
	}
}

func TestDeriveKey_DifferentUsers_DifferentKeys(t *testing.T) {
	secret := "server-secret"

	key1, err := DeriveKey("user-001", secret)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	key2, err := DeriveKey("user-002", secret)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("Different users should have different keys")
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	userId := "user-001"
	secret := "server-secret"

	key1, err := DeriveKey(userId, secret)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	key2, err := DeriveKey(userId, secret)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("Same inputs should produce same key")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "simple text",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "empty",
			plaintext: []byte{},
		},
		{
			name:      "single byte",
			plaintext: []byte{0x42},
		},
		{
			name:      "large data",
			plaintext: bytes.Repeat([]byte("A"), 1024*1024), // 1 MB
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd},
		},
		{
			name:      "unicode",
			plaintext: []byte("你好世界🌍"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Ciphertext should be longer than plaintext (nonce + tag)
			if len(ciphertext) < len(tt.plaintext)+NonceSize+16 {
				t.Errorf("Ciphertext too short: got %d, want at least %d",
					len(ciphertext), len(tt.plaintext)+NonceSize+16)
			}

			decrypted, err := Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted data doesn't match original")
			}
		})
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	key := make([]byte, KeySize)
	plaintext := []byte("Same plaintext")

	ciphertext1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	ciphertext2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Same plaintext should produce different ciphertext due to random nonce
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, KeySize)
	key2 := make([]byte, KeySize)
	key2[0] = 0xff // Different key

	plaintext := []byte("Secret message")

	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err != ErrDecryptionFailed {
		t.Errorf("Expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, KeySize)
	plaintext := []byte("Secret message")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Tamper with ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xff

	_, err = Decrypt(ciphertext, key)
	if err != ErrDecryptionFailed {
		t.Errorf("Expected ErrDecryptionFailed for tampered data, got %v", err)
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := make([]byte, KeySize)

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "empty",
			ciphertext: []byte{},
		},
		{
			name:       "just nonce",
			ciphertext: make([]byte, NonceSize),
		},
		{
			name:       "nonce + partial tag",
			ciphertext: make([]byte, NonceSize+8),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, key)
			if err != ErrInvalidCiphertext {
				t.Errorf("Expected ErrInvalidCiphertext, got %v", err)
			}
		})
	}
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := Encrypt([]byte("test"), key)
			if err == nil {
				t.Error("Expected error for invalid key size")
			}
		})
	}
}

func TestDecrypt_InvalidKeySize(t *testing.T) {
	key := make([]byte, 16) // Wrong size
	ciphertext := make([]byte, NonceSize+32)

	_, err := Decrypt(ciphertext, key)
	if err == nil {
		t.Error("Expected error for invalid key size")
	}
}

// Benchmark tests
func BenchmarkEncrypt_1KB(b *testing.B) {
	key := make([]byte, KeySize)
	plaintext := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, key)
	}
}

func BenchmarkDecrypt_1KB(b *testing.B) {
	key := make([]byte, KeySize)
	plaintext := make([]byte, 1024)
	ciphertext, _ := Encrypt(plaintext, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(ciphertext, key)
	}
}

func BenchmarkEncrypt_1MB(b *testing.B) {
	key := make([]byte, KeySize)
	plaintext := make([]byte, 1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, key)
	}
}

func BenchmarkDecrypt_1MB(b *testing.B) {
	key := make([]byte, KeySize)
	plaintext := make([]byte, 1024*1024)
	ciphertext, _ := Encrypt(plaintext, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(ciphertext, key)
	}
}

// Test user isolation - critical security test
func TestUserIsolation(t *testing.T) {
	secret := "server-secret"
	userA := "user-alice"
	userB := "user-bob"

	keyA, _ := DeriveKey(userA, secret)
	keyB, _ := DeriveKey(userB, secret)

	plaintext := []byte("Alice's private data")

	// Alice encrypts her data
	ciphertext, err := Encrypt(plaintext, keyA)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Bob cannot decrypt Alice's data
	_, err = Decrypt(ciphertext, keyB)
	if err != ErrDecryptionFailed {
		t.Error("Bob should not be able to decrypt Alice's data")
	}

	// Alice can decrypt her own data
	decrypted, err := Decrypt(ciphertext, keyA)
	if err != nil {
		t.Fatalf("Alice should be able to decrypt her data: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted data doesn't match original")
	}
}
