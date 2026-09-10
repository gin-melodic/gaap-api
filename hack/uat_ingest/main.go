// Command uat_ingest writes the Test User accounts and transaction history
// from consume_records.xlsx into the UAT environment via the ALE API.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	accountv1 "gaap-api/api/account/v1"
	authv1 "gaap-api/api/auth/v1"
	basemsg "gaap-api/api/base"
	transactionv1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/crypto"

	"google.golang.org/protobuf/proto"
)

var client = &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

type session struct {
	token     string
	sessionKey string
}

func mustKey(hexStr string) []byte {
	k, err := crypto.HexToBytes(hexStr)
	if err != nil {
		panic(err)
	}
	return k
}

func alePost(url, hexKey, auth string, body proto.Message) (int, []byte, error) {
	plain, err := proto.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	enc, err := crypto.EncryptWithHexKey(plain, hexKey)
	if err != nil {
		return 0, nil, err
	}
	iv := enc[:crypto.NonceSize]
	ct := enc[crypto.NonceSize:]
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nb := make([]byte, 8)
	rand.Read(nb)
	nonce := hex.EncodeToString(nb)
	sig := crypto.SignHMAC(crypto.BuildSignaturePayload(iv, ct, ts, nonce), mustKey(hexKey))

	req, err := http.NewRequest("POST", url, bytes.NewReader(enc))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

func decryptResp(status int, data []byte, hexKey string) string {
	if len(data) >= crypto.NonceSize+16 && hexKey != "" {
		if plain, err := crypto.DecryptWithHexKey(data, hexKey); err == nil {
			var e basemsg.ErrorResponse
			if uerr := proto.Unmarshal(plain, &e); uerr == nil {
				return fmt.Sprintf("error_proto: %s", e.String())
			}
			return fmt.Sprintf("decrypted: %.300s", plain)
		}
	}
	return fmt.Sprintf("status=%d raw=%.300q", status, data)
}

// readTSV reads a TSV file into a slice of rows.
func readTSV(path string) [][]string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("READ_ERR:", err)
		os.Exit(1)
	}
	var rows [][]string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows
}

// moneyFrom parses "1234.56" into units+nanos.
func moneyFrom(s string) (int64, int32) {
	dot := strings.IndexByte(s, '.')
	var units int64
	var nanos int32
	if dot < 0 {
		units, _ = strconv.ParseInt(s, 10, 64)
	} else {
		units, _ = strconv.ParseInt(s[:dot], 10, 64)
		frac := s[dot+1:]
		if len(frac) > 9 {
			frac = frac[:9]
		}
		for len(frac) < 9 {
			frac += "0"
		}
		n, _ := strconv.ParseInt(frac, 10, 32)
		nanos = int32(n)
	}
	return units, nanos
}

func main() {
	baseURL := os.Getenv("INGEST_BASE_URL")
	if baseURL == "" {
		baseURL = "https://gaap.local/api"
	}
	bootstrapKey := os.Getenv("ALE_BOOTSTRAP_KEY")
	if bootstrapKey == "" {
		fmt.Println("ALE_BOOTSTRAP_KEY is required")
		os.Exit(1)
	}
	email := os.Getenv("TEST_USER_EMAIL")
	password := os.Getenv("TEST_USER_PASSWORD")
	dataDir := os.Getenv("INGEST_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	skipFirstTransaction := os.Getenv("INGEST_SKIP_FIRST") == "1"

	// 1) login (retry: Cloudflare siteverify calls can be flaky)
	var status int
	var data []byte
	var err error
	for attempt := 1; attempt <= 6; attempt++ {
		status, data, err = alePost(baseURL+"/v1/auth/login", bootstrapKey, "", &authv1.LoginReq{
			Email:               email,
			Password:            password,
			CfTurnstileResponse: "uat-ingest-token",
		})
		if err == nil && status == 200 {
			break
		}
		fmt.Printf("LOGIN_RETRY %d/6 (status=%d err=%v)\n", attempt, status, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		fmt.Println("LOGIN_ERR:", err)
		os.Exit(1)
	}
	if status != 200 {
		fmt.Println("LOGIN_FAILED:", decryptResp(status, data, bootstrapKey))
		os.Exit(1)
	}
	plain, derr := crypto.DecryptWithHexKey(data, bootstrapKey)
	if derr != nil {
		fmt.Println("LOGIN_DECRYPT_ERR:", derr)
		os.Exit(1)
	}
	var loginRes authv1.LoginRes
	if uerr := proto.Unmarshal(plain, &loginRes); uerr != nil {
		fmt.Println("LOGIN_UNMARSHAL_ERR:", uerr)
		os.Exit(1)
	}
	sess := session{
		token:     loginRes.GetAuth().GetAccessToken(),
		sessionKey: loginRes.GetAuth().GetSessionKey(),
	}
	if sess.token == "" || sess.sessionKey == "" {
		fmt.Println("LOGIN_MISSING_TOKEN_OR_SESSION_KEY")
		os.Exit(1)
	}
	fmt.Println("LOGIN_OK userId=" + loginRes.GetAuth().GetUser().GetId())

	// 2) list accounts
	status, data, err = alePost(baseURL+"/v1/account/list-accounts", sess.sessionKey, sess.token, &accountv1.ListAccountsReq{
		Query: &accountv1.AccountQuery{Page: 1, Limit: 200},
	})
	if err != nil {
		fmt.Println("LIST_ACCOUNTS_ERR:", err)
		os.Exit(1)
	}
	if status != 200 {
		fmt.Println("LIST_ACCOUNTS_FAILED:", decryptResp(status, data, sess.sessionKey))
		os.Exit(1)
	}
	plain, _ = crypto.DecryptWithHexKey(data, sess.sessionKey)
	var listRes accountv1.ListAccountsRes
	if uerr := proto.Unmarshal(plain, &listRes); uerr != nil {
		fmt.Println("LIST_ACCOUNTS_UNMARSHAL_ERR:", uerr)
		os.Exit(1)
	}
	byName := make(map[string]string) // name -> id
	for _, a := range listRes.GetData() {
		byName[a.GetName()] = a.GetId()
	}
	fmt.Printf("EXISTING_ACCOUNTS(%d):", len(byName))
	for name := range byName {
		fmt.Printf(" %q", name)
	}
	fmt.Println()

	// 3) create missing category accounts
	created := 0
	for _, row := range readTSV(dataDir + "/category_accounts.tsv") {
		name, ccy, kind := row[0], row[1], row[2]
		if _, ok := byName[name]; ok {
			continue
		}
		typ := basemsg.AccountType_ACCOUNT_TYPE_EXPENSE
		if kind == "income" {
			typ = basemsg.AccountType_ACCOUNT_TYPE_INCOME
		}
		units, nanos := moneyFrom("0")
		status, data, err := alePost(baseURL+"/v1/account/create-account", sess.sessionKey, sess.token, &accountv1.CreateAccountReq{
			Input: &accountv1.AccountInput{
				Name:    name,
				Type:    typ,
				Balance: &basemsg.Money{CurrencyCode: ccy, Units: units, Nanos: nanos},
				Date:    time.Now().Format("2006-01-02"),
			},
		})
		if err != nil {
			fmt.Println("CREATE_ACCOUNT_ERR:", name, err)
			os.Exit(1)
		}
		if status != 200 {
			fmt.Println("CREATE_ACCOUNT_FAILED:", name, decryptResp(status, data, sess.sessionKey))
			os.Exit(1)
		}
		p, _ := crypto.DecryptWithHexKey(data, sess.sessionKey)
		var caRes accountv1.CreateAccountRes
		if uerr := proto.Unmarshal(p, &caRes); uerr == nil {
			byName[name] = caRes.GetAccount().GetId()
		}
		created++
	}
	fmt.Println("CATEGORY_ACCOUNTS_CREATED:", created)

	// 4) create transactions
	idToName := make(map[string]string)
	for _, row := range readTSV(dataDir + "/assets.tsv") {
		idToName[row[0]] = row[1] // account id -> name
	}

	// fetch existing transactions to dedupe (idempotent reruns)
	// server returns dates like "2026-08-03T13:43:26+08:00"; normalize to "2006-01-02 15:04:05"
	normDate := func(s string) string {
		if i := strings.IndexAny(s, "T+"); i >= 0 {
			s = s[:i]
		}
		return strings.ReplaceAll(s, "T", " ")
	}
	existing := make(map[string]struct{})
	status, data, err = alePost(baseURL+"/v1/transaction/list-transactions", sess.sessionKey, sess.token, &transactionv1.ListTransactionsReq{
		Query: &transactionv1.TransactionQuery{Page: 1, Limit: 500, StartDate: "2026-08-01", EndDate: "2026-09-09"},
	})
	if err != nil || status != 200 {
		fmt.Println("WARN: could not fetch existing transactions for dedup; proceeding without")
	} else if p, perr := crypto.DecryptWithHexKey(data, sess.sessionKey); perr == nil {
		var ltr transactionv1.ListTransactionsRes
		if proto.Unmarshal(p, &ltr) == nil {
			for _, t := range ltr.GetData() {
				a := t.GetAmount()
				key := fmt.Sprintf("%s|%s|%s|%d|%d", normDate(t.GetDate()), t.GetNote(), a.GetCurrencyCode(), a.GetUnits(), a.GetNanos())
				existing[key] = struct{}{}
			}
			fmt.Println("EXISTING_TRANSACTIONS_IN_WINDOW:", len(existing))
		}
	}

	// optional: delete duplicated transactions, keeping the earliest copy of each
	if os.Getenv("INGEST_CLEANUP") == "1" {
		var all []*transactionv1.Transaction
		page := int32(1)
		for {
			st, d, e := alePost(baseURL+"/v1/transaction/list-transactions", sess.sessionKey, sess.token, &transactionv1.ListTransactionsReq{
				Query: &transactionv1.TransactionQuery{Page: page, Limit: 200, StartDate: "2026-08-01", EndDate: "2026-09-09", SortBy: "date", SortOrder: "asc"},
			})
			if e != nil || st != 200 {
				fmt.Println("CLEANUP_LIST_ERR:", st, e)
				break
			}
			p, _ := crypto.DecryptWithHexKey(d, sess.sessionKey)
			var ltr transactionv1.ListTransactionsRes
			if proto.Unmarshal(p, &ltr) != nil {
				break
			}
			all = append(all, ltr.GetData()...)
			if int32(len(ltr.GetData())) < 200 {
				break
			}
			page++
		}
		best := make(map[string]*transactionv1.Transaction)
		for _, t := range all {
			a := t.GetAmount()
			key := fmt.Sprintf("%s|%s|%s|%d|%d|%s|%s", normDate(t.GetDate()), t.GetNote(), a.GetCurrencyCode(), a.GetUnits(), a.GetNanos(), t.GetFrom(), t.GetTo())
			cur, ok := best[key]
			if !ok || (t.GetCreatedAt() != nil && t.GetCreatedAt().AsTime().Before(cur.GetCreatedAt().AsTime())) {
				best[key] = t
			}
		}
		keepIds := make(map[string]struct{})
		for _, t := range best {
			keepIds[t.GetId()] = struct{}{}
		}
		deleted, delFailed := 0, 0
		for _, t := range all {
			if _, ok := keepIds[t.GetId()]; ok {
				continue
			}
			st, d, e := alePost(baseURL+"/v1/transaction/delete-transaction", sess.sessionKey, sess.token, &transactionv1.DeleteTransactionReq{Id: t.GetId()})
			if e != nil || st != 200 {
				fmt.Println("DELETE_FAILED:", t.GetId(), st, e, string(d))
				delFailed++
				continue
			}
			deleted++
		}
		fmt.Printf("CLEANUP_DONE total=%d keep=%d deleted=%d failed=%d\n", len(all), len(best), deleted, delFailed)
		return
	}

	txnRows := readTSV(dataDir + "/transactions.tsv")
	if skipFirstTransaction && len(txnRows) > 0 {
		txnRows = txnRows[1:]
	}
	ok, failed := 0, 0
	for _, row := range txnRows {
		seq, dt, acctId, dir, ccy, amount, cat, desc := row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7]
		acctName := idToName[acctId]

		var (
			typ          basemsg.TransactionType
			fromAccount  string
			toAccount    string
		)
		if dir == "收入" {
			typ = basemsg.TransactionType_TRANSACTION_TYPE_INCOME
			fromAccount = "工资薪金"
			toAccount = acctName
		} else {
			typ = basemsg.TransactionType_TRANSACTION_TYPE_EXPENSE
			fromAccount = acctName
			toAccount = cat
		}
		fromId, okF := byName[fromAccount]
		toId, okT := byName[toAccount]
		if !okF || !okT {
			fmt.Printf("TXN_SKIP %s missing account (from=%q to=%q)\n", seq, fromAccount, toAccount)
			failed++
			continue
		}
		units, nanos := moneyFrom(amount)
		norm := func(s string) string { return strings.ReplaceAll(s, "T", " ") }
		key := fmt.Sprintf("%s|%s|%s|%d|%d", norm(dt), desc, ccy, units, nanos)
		if _, done := existing[key]; done {
			fmt.Printf("TXN_EXISTING %s\n", seq)
			continue
		}
		status, data, err := alePost(baseURL+"/v1/transaction/create-transaction", sess.sessionKey, sess.token, &transactionv1.CreateTransactionReq{
			Input: &transactionv1.TransactionInput{
				Date:   dt,
				From:   fromId,
				To:     toId,
				Amount: &basemsg.Money{CurrencyCode: ccy, Units: units, Nanos: nanos},
				Note:   desc,
				Type:   typ,
			},
		})
		if err != nil {
			fmt.Println("TXN_ERR:", seq, err)
			failed++
			continue
		}
		if status != 200 {
			fmt.Printf("TXN_FAILED %s %s: %s\n", seq, amount, decryptResp(status, data, sess.sessionKey))
			failed++
			continue
		}
		ok++
		if ok%20 == 0 {
			fmt.Printf("TXN_PROGRESS %d/%d\n", ok, len(txnRows))
		}
	}
	fmt.Printf("TRANSACTIONS_DONE ok=%d failed=%d total_attempted=%d\n", ok, failed, len(txnRows))

	// 5) verify: list transactions
	status, data, err = alePost(baseURL+"/v1/transaction/list-transactions", sess.sessionKey, sess.token, &transactionv1.ListTransactionsReq{
		Query: &transactionv1.TransactionQuery{Page: 1, Limit: 1, StartDate: "2026-08-01", EndDate: "2026-09-09"},
	})
	if err == nil && status == 200 {
		p, _ := crypto.DecryptWithHexKey(data, sess.sessionKey)
		var ltr transactionv1.ListTransactionsRes
		if proto.Unmarshal(p, &ltr) == nil {
			j, _ := json.Marshal(ltr.GetPagination())
			fmt.Println("VERIFY_TXN_COUNT window 2026-08-01..09:", string(j), "returned_rows:", len(ltr.GetData()))
		}
	} else if err == nil {
		fmt.Println("VERIFY_FAILED:", decryptResp(status, data, sess.sessionKey))
	}
}
