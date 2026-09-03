package ale

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"gaap-api/internal/crypto"
)

func TestValidateTimestamp(t *testing.T) {
	if err := ValidateTimestamp(strconv.FormatInt(time.Now().UnixMilli(), 10)); err != nil {
		t.Fatalf("current timestamp rejected: %v", err)
	}
	if err := ValidateTimestamp("not-a-timestamp"); err == nil {
		t.Fatal("invalid timestamp format was accepted")
	}
	tooOld := strconv.FormatInt(time.Now().Add(-TimestampTolerance-time.Second).UnixMilli(), 10)
	if err := ValidateTimestamp(tooOld); err == nil {
		t.Fatal("expired timestamp was accepted")
	}
	tooFarFuture := strconv.FormatInt(time.Now().Add(TimestampTolerance+time.Second).UnixMilli(), 10)
	if err := ValidateTimestamp(tooFarFuture); err == nil {
		t.Fatal("future timestamp outside tolerance was accepted")
	}
}

func TestVerifySignatureRejectsTampering(t *testing.T) {
	hexKey := strings.Repeat("ab", 32)
	key, err := crypto.HexToBytes(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	iv := []byte("123456789012")
	ciphertext := []byte("encrypted-protobuf")
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce := "request-nonce"
	signature := crypto.SignHMAC(crypto.BuildSignaturePayload(iv, ciphertext, timestamp, nonce), key)

	valid, err := VerifySignature(iv, ciphertext, timestamp, nonce, signature, hexKey)
	if err != nil || !valid {
		t.Fatalf("valid signature rejected: valid=%t err=%v", valid, err)
	}
	valid, err = VerifySignature(iv, []byte("tampered"), timestamp, nonce, signature, hexKey)
	if err != nil {
		t.Fatalf("tampered signature returned unexpected error: %v", err)
	}
	if valid {
		t.Fatal("tampered ciphertext signature was accepted")
	}
}

func TestSessionKeysAreIsolatedBySession(t *testing.T) {
	t.Setenv("GF_ENV", "test")
	t.Setenv("ENV", "")
	ctx := context.Background()
	userID := "user-a"

	first, err := GenerateAndStoreSessionKey(ctx, userID, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateAndStoreSessionKey(ctx, userID, "session-2")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two sessions received the same random key")
	}

	if err := InvalidateSessionKey(ctx, userID, "session-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetSessionKey(ctx, userID, "session-1"); err == nil {
		t.Fatal("invalidated session key is still available")
	}
	got, err := GetSessionKey(ctx, userID, "session-2")
	if err != nil {
		t.Fatalf("independent session was invalidated: %v", err)
	}
	if got != second {
		t.Fatal("independent session key changed")
	}
	_ = InvalidateSessionKey(ctx, userID, "session-2")
}

func TestProductionDisablesMemoryFallback(t *testing.T) {
	t.Setenv("GF_ENV", " Production ")
	t.Setenv("ENV", "")
	if allowInMemoryFallback() {
		t.Fatal("production environment allowed in-memory ALE fallback")
	}
}
