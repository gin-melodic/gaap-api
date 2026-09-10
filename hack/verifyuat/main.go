package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"crypto/tls"
	"net/http"
	"os"
	"strconv"
	"time"

	accountv1 "gaap-api/api/account/v1"
	authv1 "gaap-api/api/auth/v1"
	"gaap-api/api/base"
	"gaap-api/internal/crypto"

	"google.golang.org/protobuf/proto"
)

var client = &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

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

func mustKey(hexStr string) []byte {
	k, err := crypto.HexToBytes(hexStr)
	if err != nil {
		panic(err)
	}
	return k
}

func decryptResp(status int, data []byte, hexKey string) (string, error) {
	if status == 200 && len(data) >= crypto.NonceSize+16 {
		plain, err := crypto.DecryptWithHexKey(data, hexKey)
		if err != nil {
			return fmt.Sprintf("status=200 but decrypt failed: %v", err), err
		}
		return string(plain), nil
	}
	// error path: try encrypted ErrorResponse, else raw bytes
	if len(data) >= crypto.NonceSize+16 && hexKey != "" {
		if plain, err := crypto.DecryptWithHexKey(data, hexKey); err == nil {
			var e base.ErrorResponse
			if uerr := proto.Unmarshal(plain, &e); uerr == nil {
				return fmt.Sprintf("status=%d error_proto: %s", status, e.String()), nil
			}
			return fmt.Sprintf("status=%d decrypted: %.300s", status, plain), nil
		}
	}
	return fmt.Sprintf("status=%d raw=%.300q", status, data), nil
}

func main() {
	baseURL := os.Getenv("VERIFY_BASE_URL") // e.g. https://gaap.local/api
	bootstrapKey := os.Getenv("ALE_BOOTSTRAP_KEY")
	groupId := os.Args[1]

	// 1) demo-login (bootstrap key, empty request proto)
	status, data, err := alePost(baseURL+"/v1/auth/demo-login", bootstrapKey, "", &authv1.DemoLoginReq{})
	if err != nil {
		fmt.Println("LOGIN_ERR:", err)
		os.Exit(1)
	}
	var loginRes authv1.LoginRes
	if status == 200 {
		plain, derr := crypto.DecryptWithHexKey(data, bootstrapKey)
		if derr != nil {
			fmt.Println("LOGIN_DECRYPT_ERR:", derr)
			os.Exit(1)
		}
		if uerr := proto.Unmarshal(plain, &loginRes); uerr != nil {
			fmt.Println("LOGIN_UNMARSHAL_ERR:", uerr)
			os.Exit(1)
		}
	} else {
		msg, _ := decryptResp(status, data, bootstrapKey)
		fmt.Println("LOGIN_FAILED:", msg)
		os.Exit(1)
	}
	token := loginRes.GetAuth().GetAccessToken()
	sessionKey := loginRes.GetAuth().GetSessionKey()
	if token == "" || sessionKey == "" {
		fmt.Println("LOGIN_MISSING_TOKEN_OR_SESSION_KEY")
		os.Exit(1)
	}
	fmt.Println("LOGIN_OK userId=" + loginRes.GetAuth().GetUser().GetId())

	// 2) delete-account (session key)
	status, data, err = alePost(baseURL+"/v1/account/delete-account", sessionKey, token, &accountv1.DeleteAccountReq{Id: groupId})
	if err != nil {
		fmt.Println("DELETE_ERR:", err)
		os.Exit(1)
	}
	if status == 200 {
		plain, derr := crypto.DecryptWithHexKey(data, sessionKey)
		if derr != nil {
			fmt.Println("DELETE_DECRYPT_ERR:", derr)
			os.Exit(1)
		}
		var delRes accountv1.DeleteAccountRes
		if uerr := proto.Unmarshal(plain, &delRes); uerr != nil {
			fmt.Println("DELETE_UNMARSHAL_ERR:", uerr)
			os.Exit(1)
		}
		fmt.Println("DELETE_OK taskId=" + delRes.GetTaskId())
		return
	}
	msg, _ := decryptResp(status, data, sessionKey)
	fmt.Println("DELETE_FAILED:", msg)
	os.Exit(1)
}
