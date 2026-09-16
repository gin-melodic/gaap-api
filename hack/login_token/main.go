// Command login_token performs an ALE login and prints the access token,
// for ad-hoc ops smoke tests against UAT.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	authv1 "gaap-api/api/auth/v1"
	"gaap-api/internal/crypto"

	"google.golang.org/protobuf/proto"
)

var client = &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

func mustKey(hexStr string) []byte {
	k, err := crypto.HexToBytes(hexStr)
	if err != nil {
		panic(err)
	}
	return k
}

func alePost(url, hexKey string, body proto.Message) (int, []byte, error) {
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
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

func main() {
	baseURL := os.Getenv("TOKEN_BASE_URL")
	if baseURL == "" {
		baseURL = "https://gaap.local/api"
	}
	bootstrapKey := os.Getenv("ALE_BOOTSTRAP_KEY")
	email := os.Getenv("LOGIN_EMAIL")
	password := os.Getenv("LOGIN_PASSWORD")
	if bootstrapKey == "" || email == "" || password == "" {
		fmt.Println("ALE_BOOTSTRAP_KEY, LOGIN_EMAIL, LOGIN_PASSWORD are required")
		os.Exit(1)
	}

	for attempt := 1; attempt <= 6; attempt++ {
		status, data, err := alePost(baseURL+"/v1/auth/login", bootstrapKey, &authv1.LoginReq{
			Email:               email,
			Password:            password,
			CfTurnstileResponse: "login-token-smoke",
		})
		if err != nil {
			fmt.Println("LOGIN_ERR:", err)
			os.Exit(1)
		}
		if status != 200 {
			fmt.Printf("LOGIN_RETRY %d/6 status=%d raw=%.200q\n", attempt, status, data)
			time.Sleep(2 * time.Second)
			continue
		}
		plain, derr := crypto.DecryptWithHexKey(data, bootstrapKey)
		if derr != nil {
			fmt.Println("LOGIN_DECRYPT_ERR:", derr)
			os.Exit(1)
		}
		var res authv1.LoginRes
		if uerr := proto.Unmarshal(plain, &res); uerr != nil {
			fmt.Println("LOGIN_UNMARSHAL_ERR:", uerr)
			os.Exit(1)
		}
		token := res.GetAuth().GetAccessToken()
		if token == "" {
			fmt.Println("LOGIN_MISSING_TOKEN")
			os.Exit(1)
		}
		fmt.Println("TOKEN=" + token)
		return
	}
	fmt.Println("LOGIN_FAILED after retries")
	os.Exit(1)
}
