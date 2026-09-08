package middleware

// Wire-level probe for the UAT ALE error channel (DEF-028). Gated by env so it
// never runs in normal test passes:
//
//	GAAP_UAT_PROBE=1 GAAP_UAT_BASE=https://gaap.local/api \
//	  ALE_BOOTSTRAP_KEY=<from .env.uat> \
//	  GAAP_UAT_USD_ID=<account id> GAAP_UAT_EUR_ID=<account id> \
//	go test ./internal/middleware/ -run TestALEUATProbe -v

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	authv1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	txnv1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/crypto"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func TestALEUATProbe(t *testing.T) {
	if os.Getenv("GAAP_UAT_PROBE") != "1" {
		t.Skip("set GAAP_UAT_PROBE=1 to run the live UAT ALE probe")
	}
	baseURL := envOr("GAAP_UAT_BASE", "https://gaap.local/api")
	bootstrapKey := os.Getenv("ALE_BOOTSTRAP_KEY")
	if bootstrapKey == "" {
		t.Fatal("ALE_BOOTSTRAP_KEY not set")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // local UAT self-signed cert
			DisableCompression: true, // observe raw wire bytes like a non-decoding client
		},
		Timeout: 30 * time.Second,
	}

	login, err := demoLoginRaw(t, client, baseURL, bootstrapKey)
	if err != nil {
		t.Fatalf("demo login: %v", err)
	}
	sessionKey := login.Auth.SessionKey
	t.Logf("session key (first 8 hex): %s…", sessionKey[:8])

	cases := []struct {
		name string
		from uuid.UUID
		to   uuid.UUID
		want string
	}{
		{"cross-currency transfer (USD->EUR)", mustUUID(t, "GAAP_UAT_USD_ID"), mustUUID(t, "GAAP_UAT_EUR_ID"), "account currency mismatch"},
		{"same-account transfer (USD->USD)", mustUUID(t, "GAAP_UAT_USD_ID"), mustUUID(t, "GAAP_UAT_USD_ID"), "source and destination accounts must differ"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := &txnv1.TransactionInput{
				Date:   "2026-09-08",
				From:   tc.from.String(),
				To:     tc.to.String(),
				Amount: &base.Money{CurrencyCode: "USD", Units: 1, Nanos: 0},
				Type:   base.TransactionType_TRANSACTION_TYPE_TRANSFER,
			}
			body := protoMustMarshal(t, &txnv1.CreateTransactionReq{Input: input})

			for _, variant := range []string{"raw", "browser-like"} {
				t.Run(variant, func(t *testing.T) {
					status, headers, payload := postALEWithAuth(t, client, baseURL+"/transaction/create-transaction", body, sessionKey, alePostOpts{
						accessToken: login.Auth.AccessToken,
						browserLike: variant == "browser-like",
					})
					printProbeResponse(t, status, headers, payload)

					if status < 400 {
						t.Fatalf("expected 4xx rejection, got %d", status)
					}
					if headers.Get("X-ALE-Encrypted") != "1" {
						t.Logf("NOTE: X-ALE-Encrypted header not set; body may be plaintext error envelope")
					}

					plaintext, err := crypto.Decrypt(payload, mustHex(t, sessionKey))
					if err != nil {
						// Try bootstrap key as well to distinguish wrong-key vs wrong-envelope.
						p2, err2 := crypto.Decrypt(payload, mustHex(t, bootstrapKey))
						t.Errorf("FAILED to decrypt error response with SESSION key: %v (bootstrap attempt: %v, len=%d)", err, err2, len(p2))
						return
					}

					respErr := &base.ErrorResponse{}
					if derr := proto.Unmarshal(plaintext, respErr); derr != nil {
						t.Errorf("decrypted payload is not an ErrorResponse protobuf: %v; raw=% x", derr, truncate(plaintext, 64))
						return
					}
					t.Logf("DECRYPTED OK -> code=%d message=%q request_id=%s", respErr.Code, respErr.Message, respErr.RequestId)
					if !strings.Contains(respErr.Message, tc.want) {
						t.Errorf("message %q does not contain expected %q", respErr.Message, tc.want)
					}
				})
			}
		})
	}
}

// demoLoginRaw performs /auth/demo-login over ALE with the bootstrap key and
// returns the decoded LoginRes.
func demoLoginRaw(t *testing.T, client *http.Client, baseURL, bootstrapKey string) (*authv1.LoginRes, error) {
	t.Helper()
	var empty authv1.DemoLoginReq
	body := protoMustMarshal(t, &empty)

	status, headers, payload := postALE(t, client, baseURL+"/auth/demo-login", body, bootstrapKey, false)
	printProbeResponse(t, status, headers, payload)
	t.Logf("login full body hex: %x", payload)
	if status != 200 {
		return nil, fmt.Errorf("demo login returned %d; body=% x", status, truncate(payload, 64))
	}
	plaintext, err := crypto.Decrypt(payload, mustHex(t, bootstrapKey))
	if err != nil {
		return nil, fmt.Errorf("decrypt login response: %v", err)
	}
	res := &authv1.LoginRes{}
	if err := proto.Unmarshal(plaintext, res); err != nil {
		return nil, fmt.Errorf("unmarshal LoginRes: %v", err)
	}
	t.Logf("demo login OK; access token present=%v session key length=%d", res.Auth.GetAccessToken() != "", len(res.Auth.SessionKey))
	return res, nil
}

// postALE encrypts body with hexKey (session or bootstrap), POSTs it to url and
// returns status + headers + raw response bytes. browserLike=true adds the
// Accept-Encoding header a real browser sends (response stays undecoded).
type alePostOpts struct {
	accessToken string
	browserLike bool
}

func postALE(t *testing.T, client *http.Client, url string, body []byte, hexKey string, browserLike bool) (int, http.Header, []byte) {
	return postALEWithAuth(t, client, url, body, hexKey, alePostOpts{browserLike: browserLike})
}

func postALEWithAuth(t *testing.T, client *http.Client, url string, body []byte, hexKey string, opts alePostOpts) (int, http.Header, []byte) {
	t.Helper()

	key := mustHex(t, hexKey)
	envelope, err := crypto.Encrypt(body, key) // IV (12 bytes) || ciphertext || tag
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	iv, ciphertext := envelope[:crypto.NonceSize], envelope[crypto.NonceSize:]

	fullBody := append([]byte{}, envelope...)
	nonce := uuid.NewString()
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	realSignature := crypto.SignHMAC(crypto.BuildSignaturePayload(iv, ciphertext, timestamp, nonce), key)

	req, err := http.NewRequest("POST", url, bytes.NewReader(fullBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Signature", realSignature)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Nonce", nonce)
	if opts.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.accessToken)
	}
	if opts.browserLike {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, payload
}

func printProbeResponse(t *testing.T, status int, headers http.Header, payload []byte) {
	t.Helper()
	t.Logf("HTTP %d len=%d content-type=%q content-encoding=%q ale-encrypted=%q content-length=%q",
		status, len(payload), headers.Get("Content-Type"), headers.Get("Content-Encoding"), headers.Get("X-ALE-Encrypted"), headers.Get("Content-Length"))
	if len(payload) > 0 {
		t.Logf("body head: % x", truncate(payload, 48))
	}
	for _, h := range []string{"Server", "Via"} {
		if v := headers.Get(h); v != "" {
			t.Logf("header %s=%q", h, v)
		}
	}
	// Detect gzip/zstd magic so a compressed body is obvious.
	switch {
	case len(payload) > 2 && payload[0] == 0x1f && payload[1] == 0x8b:
		t.Logf("body looks GZIP-compressed on the wire")
	case len(payload) >= 4 && bytes.Equal(payload[:4], []byte{0x28, 0xb5, 0x2f, 0xfd}):
		t.Logf("body looks ZSTD-compressed on the wire")
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func mustUUID(t *testing.T, name string) uuid.UUID {
	t.Helper()
	v := os.Getenv(name)
	id, err := uuid.Parse(v)
	if err != nil {
		t.Fatalf("%s invalid: %v", name, err)
	}
	return id
}

func mustHex(t *testing.T, hexKey string) []byte {
	t.Helper()
	b, err := crypto.HexToBytes(hexKey)
	if err != nil {
		t.Fatalf("hex key: %v", err)
	}
	return b
}

func protoMustMarshal(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}
