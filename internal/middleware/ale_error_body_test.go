package middleware

import (
	"testing"

	"gaap-api/api/base"
	"gaap-api/internal/ale"
	"gaap-api/internal/crypto"

	"google.golang.org/protobuf/proto"
)

// TestALEErrorResponseRoundTrip guards the DEF-028 regression: an encrypted ALE
// error body must be exactly IV(12)||ciphertext||tag of a decodable ErrorResponse,
// with no leading plaintext (GoFrame's WriteStatus used to prepend the status
// text such as "Bad Request", shifting the IV slice and failing GCM auth).
func TestALEErrorResponseRoundTrip(t *testing.T) {
	keyBytes := make([]byte, crypto.KeySize)
	for i := range keyBytes {
		keyBytes[i] = byte(i + 1) // deterministic non-zero key material
	}
	hexKey := crypto.BytesToHex(keyBytes)

	tests := []struct {
		status  int
		message string
	}{
		{status: 400, message: "account currency mismatch"},
		{status: 400, message: "source and destination accounts must differ"},
		{status: 401, message: "invalid email or password"},
		{status: 403, message: "registration unavailable"},
		{status: 404, message: "transaction not found"},
		{status: 500, message: "internal server error"},
	}

	for _, tc := range tests {
		t.Run(tc.message, func(t *testing.T) {
			requestId := "test-request-id"
			body, encrypted := protoErrorBody(tc.status, tc.message, requestId, hexKey)

			if !encrypted {
				t.Fatalf("protoErrorBody returned encrypted=false for valid key")
			}
			if len(body) < crypto.NonceSize+16 {
				t.Fatalf("envelope too short to hold IV + GCM tag: %d bytes", len(body))
			}

			plaintext, err := ale.DecryptRequest(body, hexKey)
			if err != nil {
				t.Fatalf("DecryptRequest failed on error envelope: %v", err)
			}

			resp := &base.ErrorResponse{}
			if err := proto.Unmarshal(plaintext, resp); err != nil {
				t.Fatalf("decrypted payload is not a valid ErrorResponse protobuf (envelope corrupted?): %v; head=% x", err, plaintext[:min(len(plaintext), 16)])
			}
			if resp.Code != int32(tc.status) || resp.Message != tc.message || resp.RequestId != requestId {
				t.Fatalf("round-trip mismatch: got code=%d message=%q request_id=%s; want %d/%q/%q",
					resp.Code, resp.Message, resp.RequestId, tc.status, tc.message, requestId)
			}

			// Regression assertion for DEF-028: the body must not begin with a
			// plaintext status-text prefix. Dropping any leading byte has to break
			// GCM authentication if the envelope is well-formed from byte 0.
			if _, err := ale.DecryptRequest(body[1:], hexKey); err == nil {
				t.Fatalf("decryption succeeded after dropping a leading byte; IV is not at offset 0")
			}
		})
	}
}

// TestALEErrorResponsePlainPath covers the unencrypted fallback (session expired):
// the body must be plain ErrorResponse bytes with encrypted=false.
func TestALEErrorResponsePlainPath(t *testing.T) {
	body, encrypted := protoErrorBody(401, "secure session expired, please login again", "req-plain", "")
	if encrypted {
		t.Fatal("expected encrypted=false without a key")
	}

	resp := &base.ErrorResponse{}
	if err := proto.Unmarshal(body, resp); err != nil {
		t.Fatalf("plain body is not an ErrorResponse protobuf: %v", err)
	}
	if resp.Code != 401 || resp.RequestId != "req-plain" {
		t.Fatalf("unexpected plain error response: %+v", resp)
	}

	// An empty hexKey never yields encryption even for other statuses.
	if _, enc := protoErrorBody(400, "validation failed", "req-2", ""); enc {
		t.Fatal("expected encrypted=false for empty key")
	}
}
