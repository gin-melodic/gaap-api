// Env-gated live test that restores the multi-currency UAT demo data for the
// ONLINE_DEMO user (gaap_test_feedback@ginmel.ai) through the real ALE-encrypted
// API endpoints, then re-runs the DEF-028 transfer repros and asserts both real
// server validation messages come back readable.
//
//	Env vars:
//	  GAAP_UAT_SEED=1            run this test
//	  ALE_BOOTSTRAP_KEY=<hex>    same key as .env.uat (64 hex chars)
//	  GAAP_UAT_BASE              default https://gaap.local/api/v1
//
// Expected end state after a green run:
//   - mc-usd-asset          USD asset, opening balance US$100, then US$5 expense -> US$95
//   - UAT SameCcy Expense   USD expense account (balance -US$5)
//   - uat-eur-card          EUR asset, opening balance EUR 50 (+ auto equity voucher)
//   - exchange_rates:       manual USD->CNY = 7.2
package middleware

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	acctv1 "gaap-api/api/account/v1"
	base "gaap-api/api/base"
	cfgv1 "gaap-api/api/config/v1"
	txnv1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/ale"
	"gaap-api/internal/crypto"

	"google.golang.org/protobuf/proto"
)

func TestALEUATDemoSeed(t *testing.T) {
	if testing.Short() || os.Getenv("GAAP_UAT_SEED") != "1" {
		t.Skip("set GAAP_UAT_SEED=1 (plus ALE_BOOTSTRAP_KEY, optional GAAP_UAT_BASE) to run against live UAT")
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
	t.Logf("demo login OK; session key (first 8 hex): %s…", sessionKey[:8])
	opts := alePostOpts{accessToken: login.Auth.AccessToken}

	postDecrypted := func(path string, req proto.Message, out proto.Message) {
		t.Helper()
		body := protoMustMarshal(t, req)
		status, _, payload := postALEWithAuth(t, client, baseURL+path, body, sessionKey, opts)
		if status < 200 || status >= 300 {
			detail := "<unencrypted>"
			if p, err := ale.DecryptRequest(payload, sessionKey); err == nil {
				e := &base.ErrorResponse{}
				if uerr := proto.Unmarshal(p, e); uerr == nil && e.Message != "" {
					detail = e.Message
				} else if uerr == nil {
					detail = "empty ErrorResponse"
				}
			}
			t.Fatalf("POST %s -> HTTP %d: %s (raw head: % x)", path, status, detail, truncate(payload, 64))
		}
		plaintext, err := ale.DecryptRequest(payload, sessionKey)
		if err != nil {
			t.Fatalf("POST %s returned %d but session-key decryption failed: %v", path, status, err)
		}
		if err := proto.Unmarshal(plaintext, out); err != nil {
			t.Fatalf("POST %s response is not the expected protobuf message: %v; head=% x", path, err, truncate(plaintext, 32))
		}
	}

	const uatDate = "2026-09-08" // matches the live UAT round date on this stack

	// Optional maintenance step (GAAP_UAT_DELETE_TXN_ID): delete one transaction by
	// id through the real endpoint, e.g. to clean up a browser-run control txn.
	if deleteId := os.Getenv("GAAP_UAT_DELETE_TXN_ID"); deleteId != "" {
		var delRes txnv1.DeleteTransactionRes
		postDecrypted("/transaction/delete-transaction", &txnv1.DeleteTransactionReq{Id: deleteId}, &delRes)
		t.Logf("deleted transaction id=%s via API (base message %q)", deleteId, delRes.GetBase().GetMessage())
		if os.Getenv("GAAP_UAT_SEED_ONLY_DELETE") == "1" {
			t.Log("delete-only mode; skipping seed and repro steps")
			return
		}
	}

	// 1. USD asset with opening balance US$100 (UAT evidence: created 2026-09-07).
	var usdRes acctv1.CreateAccountRes
	postDecrypted("/account/create-account", &acctv1.CreateAccountReq{Input: &acctv1.AccountInput{
		Name:    "mc-usd-asset",
		Type:    base.AccountType_ACCOUNT_TYPE_ASSET,
		Balance: &base.Money{CurrencyCode: "USD", Units: 100},
		Date:    uatDate,
	}}, &usdRes)
	usdId := usdRes.GetAccount().GetId()
	if got := usdRes.GetAccount().GetBalance(); got == nil || got.GetUnits() != 100 || got.GetCurrencyCode() != "USD" {
		t.Fatalf("mc-usd-asset created with unexpected balance: %+v", got)
	}
	t.Logf("created mc-usd-asset id=%s (US$100)", usdId)

	// 2. USD expense account for the same-currency US$5 expense transaction.
	var expRes acctv1.CreateAccountRes
	postDecrypted("/account/create-account", &acctv1.CreateAccountReq{Input: &acctv1.AccountInput{
		Name:    "UAT SameCcy Expense",
		Type:    base.AccountType_ACCOUNT_TYPE_EXPENSE,
		Balance: &base.Money{CurrencyCode: "USD"}, // zero units pins the currency without an opening balance
		Date:    uatDate,
	}}, &expRes)
	expId := expRes.GetAccount().GetId()
	if got := expRes.GetAccount().GetBalance(); got == nil || got.GetCurrencyCode() != "USD" {
		t.Fatalf("UAT SameCcy Expense created with unexpected balance: %+v", got)
	}
	t.Logf("created UAT SameCcy Expense id=%s (USD)", expId)

	// 3. EUR asset card with opening balance EUR 50 (+ auto equity voucher).
	var eurRes acctv1.CreateAccountRes
	postDecrypted("/account/create-account", &acctv1.CreateAccountReq{Input: &acctv1.AccountInput{
		Name:    "uat-eur-card",
		Type:    base.AccountType_ACCOUNT_TYPE_ASSET,
		Balance: &base.Money{CurrencyCode: "EUR", Units: 50},
		Date:    uatDate,
	}}, &eurRes)
	eurId := eurRes.GetAccount().GetId()
	if got := eurRes.GetAccount().GetBalance(); got == nil || got.GetUnits() != 50 || got.GetCurrencyCode() != "EUR" {
		t.Fatalf("uat-eur-card created with unexpected balance: %+v", got)
	}
	t.Logf("created uat-eur-card id=%s (EUR 50)", eurId)

	// 4. Manual USD->CNY rate override = 7.2.
	var rateRes cfgv1.SetExchangeRateRes
	postDecrypted("/config/set-exchange-rate", &cfgv1.SetExchangeRateReq{Currency: "CNY", Rate: "7.2"}, &rateRes)
	if got := rateRes.GetRate(); got == nil || got.GetSource() != "manual" {
		t.Fatalf("manual rate not persisted as source=manual: %+v", got)
	}
	t.Logf("set manual USD->CNY = %s (source=%s)", rateRes.GetRate().GetRate(), rateRes.GetRate().GetSource())

	// 5. Same-currency US$5 expense: mc-usd-asset -> UAT SameCcy Expense (leaves US$95).
	var txRes txnv1.CreateTransactionRes
	postDecrypted("/transaction/create-transaction", &txnv1.CreateTransactionReq{Input: &txnv1.TransactionInput{
		Date:   uatDate,
		From:   usdId,
		To:     expId,
		Amount: &base.Money{CurrencyCode: "USD", Units: 5},
		Type:   base.TransactionType_TRANSACTION_TYPE_EXPENSE,
	}}, &txRes)
	if txRes.GetTransaction() == nil {
		t.Fatal("expense transaction response missing Transaction")
	}
	t.Logf("committed US$5 expense txn id=%s -> mc-usd-asset now US$95", txRes.GetTransaction().GetId())

	// 6. DEF-028 repro A: cross-currency asset transfer must surface "account currency mismatch".
	assertReadableError(t, client, baseURL, sessionKey, opts, "cross-currency USD->EUR",
		&txnv1.CreateTransactionReq{Input: &txnv1.TransactionInput{
			Date:   uatDate,
			From:   usdId,
			To:     eurId,
			Amount: &base.Money{CurrencyCode: "USD", Units: 1},
			Type:   base.TransactionType_TRANSACTION_TYPE_TRANSFER,
		}}, "account currency mismatch")

	// 7. DEF-028 repro B: same-account transfer must surface "source and destination accounts must differ".
	assertReadableError(t, client, baseURL, sessionKey, opts, "same-account USD->USD",
		&txnv1.CreateTransactionReq{Input: &txnv1.TransactionInput{
			Date:   uatDate,
			From:   usdId,
			To:     usdId,
			Amount: &base.Money{CurrencyCode: "USD", Units: 1},
			Type:   base.TransactionType_TRANSACTION_TYPE_TRANSFER,
		}}, "source and destination accounts must differ")

	t.Log("demo data restored; both DEF-028 validation messages verified over the wire")
}

// assertReadableError posts an ALE request expected to be rejected with HTTP 4xx and
// asserts the (decrypted) ErrorResponse message contains want.
func assertReadableError(t *testing.T, client *http.Client, baseURL, sessionKey string, opts alePostOpts, name string, req proto.Message, want string) {
	t.Helper()

	body := protoMustMarshal(t, req)
	status, headers, payload := postALEWithAuth(t, client, baseURL+"/transaction/create-transaction", body, sessionKey, opts)
	if status < 400 || status >= 500 {
		t.Fatalf("[%s] expected 4xx rejection, got %d", name, status)
	}

	var message string
	switch {
	case headers.Get("X-ALE-Encrypted") == "1" && len(payload) > crypto.NonceSize+crypto.HMACSize:
		plaintext, err := ale.DecryptRequest(payload, sessionKey)
		if err != nil {
			t.Fatalf("[%s] encrypted error body failed to decrypt with the session key (the DEF-028 regression): %v", name, err)
		}
		e := &base.ErrorResponse{}
		if uerr := proto.Unmarshal(plaintext, e); uerr == nil {
			message = e.Message
		} else {
			message = "<not an ErrorResponse>"
		}
	case len(payload) > 0:
		e := &base.ErrorResponse{} // unencrypted fallback (session expired / no key)
		if uerr := proto.Unmarshal(payload, e); uerr == nil && e.Message != "" {
			message = e.Message
		} else {
			message = string(truncate(payload, 120))
		}
	default:
		message = "<empty body>"
	}

	if !strings.Contains(message, want) {
		t.Fatalf("[%s] decrypted error message %q does not contain expected %q", name, message, want)
	}
	t.Logf("[%s] HTTP %d -> readable message: %q", name, status, message)
}
