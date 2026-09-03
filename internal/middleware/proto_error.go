package middleware

import (
	"gaap-api/api/base"
	"gaap-api/internal/ale"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	HeaderALEEncrypted      = "X-ALE-Encrypted"
	HeaderALESessionExpired = "X-ALE-Session-Expired"
)

func writeSessionExpiredError(r *ghttp.Request, message string) {
	// The server no longer has the per-session key, so this particular 401
	// cannot be ALE-encrypted. The explicit marker lets the client distinguish
	// it from an untrusted or accidentally unencrypted business response.
	r.Response.Header().Set(HeaderALESessionExpired, "1")
	writeProtoError(r, 401, message, "")
}

func writeProtoError(r *ghttp.Request, status int, message, hexKey string) {
	requestId := r.GetHeader("X-Request-ID")
	if requestId == "" {
		requestId = uuid.NewString()
	}

	payload, err := proto.Marshal(&base.ErrorResponse{
		Code:      int32(status),
		Message:   message,
		RequestId: requestId,
	})
	if err != nil {
		payload = nil
	}

	if hexKey != "" && len(payload) > 0 {
		if encrypted, encryptErr := ale.EncryptResponse(payload, hexKey); encryptErr == nil {
			payload = encrypted
			r.Response.Header().Set(HeaderALEEncrypted, "1")
		}
	}

	r.Response.Header().Set("Content-Type", "application/octet-stream")
	r.Response.Header().Set("X-Request-ID", requestId)
	r.Response.ClearBuffer()
	r.Response.WriteStatus(status)
	r.Response.Write(payload)
	r.Exit()
}
