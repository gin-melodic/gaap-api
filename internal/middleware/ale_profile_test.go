// Env-gated live repro for DEF-029: exercise /v1/user/update-profile through the
// real ALE session channel with exactly what the Currency Settings UI sends when
// switching the base currency (nickname + plan from the loaded profile, new
// mainCurrency), then print the decrypted response or error so the failure mode is
// readable on the wire.
//
//	Env vars:
//	  GAAP_UAT_PROFILE=1             run this test
//	  ALE_BOOTSTRAP_KEY=<hex>        same key as .env.uat (64 hex chars)
//	  GAAP_UAT_BASE                  default https://gaap.local/api/v1
//	  GAAP_UAT_NEW_CURRENCY          default USD
package middleware

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"
	"time"

	base "gaap-api/api/base"
	userv1 "gaap-api/api/user/v1"
	"gaap-api/internal/ale"

	"google.golang.org/protobuf/proto"
)

func TestALEUATUpdateProfile(t *testing.T) {
	if testing.Short() || os.Getenv("GAAP_UAT_PROFILE") != "1" {
		t.Skip("set GAAP_UAT_PROFILE=1 (plus ALE_BOOTSTRAP_KEY, optional GAAP_UAT_BASE) to run against live UAT")
	}
	baseURL := envOr("GAAP_UAT_BASE", "https://gaap.local/api/v1")
	bootstrapKey := os.Getenv("ALE_BOOTSTRAP_KEY")
	if bootstrapKey == "" {
		t.Fatal("ALE_BOOTSTRAP_KEY not set")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // local UAT self-signed cert
			DisableCompression: true,
		},
		Timeout: 30 * time.Second,
	}

	login, err := demoLoginRaw(t, client, baseURL, bootstrapKey)
	if err != nil {
		t.Fatalf("demo login: %v", err)
	}
	sessionKey := login.Auth.SessionKey
	opts := alePostOpts{accessToken: login.Auth.AccessToken}

	newCurrency := envOr("GAAP_UAT_NEW_CURRENCY", "USD")

	// Mirror the Currency Settings UI payload exactly (user.nickname='demo_user', user.plan=1, no avatar).
	reqBody := protoMustMarshal(t, &userv1.UpdateUserProfileReq{Input: &userv1.UserInput{
		Nickname:     "demo_user",
		Plan:         1,
		MainCurrency: &newCurrency,
	}})

	status, headers, payload := postALEWithAuth(t, client, baseURL+"/user/update-profile", reqBody, sessionKey, opts)
	t.Logf("HTTP %d content-type=%q ale-encrypted=%q body-head=% x", status, headers.Get("Content-Type"), headers.Get("X-ALE-Encrypted"), truncate(payload, 48))

	switch {
	case status >= 200 && status < 300:
		res := &userv1.UpdateUserProfileRes{}
		if p, derr := ale.DecryptRequest(payload, sessionKey); derr == nil {
			uerr := proto.Unmarshal(p, res)
			t.Logf("SUCCESS main_currency=%q nickname=%q plan=%d base=%q",
				res.GetUser().GetMainCurrency(), res.GetUser().GetNickname(), int(res.GetUser().GetPlan()), res.GetBase().GetMessage())
			_ = uerr
		} else {
			t.Fatalf("success status but decrypt failed: %v", derr)
		}
	case headers.Get("X-ALE-Encrypted") == "1" && len(payload) > 0:
		p, derr := ale.DecryptRequest(payload, sessionKey)
		if derr != nil {
			t.Fatalf("encrypted error body failed to decrypt: %v", derr)
		}
		e := &base.ErrorResponse{}
		if uerr := proto.Unmarshal(p, e); uerr == nil {
			t.Logf("SERVER ERROR code=%d message=%q requestId=%q", int(e.GetCode()), e.GetMessage(), e.GetRequestId())
			return
		}
		t.Fatalf("error body is not an ErrorResponse: % x", p)
	default:
		t.Fatalf("unexpected response shape; raw head: % x", truncate(payload, 128))
	}
}
