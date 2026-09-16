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

	body, encrypted := protoErrorBody(status, message, requestId, hexKey)

	r.Response.Header().Set("Content-Type", "application/octet-stream")
	r.Response.Header().Set("X-Request-ID", requestId)
	if encrypted {
		r.Response.Header().Set(HeaderALEEncrypted, "1")
	}
	// Drop anything the handler chain buffered earlier (e.g. a default JSON
	// envelope), so the body is exactly protoErrorBody's output byte-for-byte.
	r.Response.ClearBuffer()
	// WriteHeader only sets the status code. Do NOT use WriteStatus here: it
	// also writes http.StatusText(status) ("Bad Request", "Not Found", ...) as
	// literal body bytes in front of the ALE envelope, which shifts the client's
	// fixed IV/ciphertext slicing and breaks AES-GCM authentication for every
	// encrypted error response (DEF-028).
	r.Response.WriteHeader(status)
	if len(body) > 0 {
		r.Response.Write(body)
	}
	r.Exit()
}

// protoErrorBody builds the exact body written by writeProtoError. With a
// non-empty hexKey it returns the ALE envelope IV(12)||ciphertext||tag of an
// ErrorResponse protobuf and encrypted=true; otherwise (session expired / key
// unavailable) the plain ErrorResponse bytes and encrypted=false. Any failure
// yields a nil body so the transport can fall back to the default status text.
func protoErrorBody(status int, message, requestId, hexKey string) ([]byte, bool) {
	payload, err := proto.Marshal(&base.ErrorResponse{
		Code:      int32(status),
		Message:   message,
		RequestId: requestId,
	})
	if err != nil || len(payload) == 0 {
		return nil, false
	}

	if hexKey != "" {
		if encrypted, encryptErr := ale.EncryptResponse(payload, hexKey); encryptErr == nil {
			return encrypted, true
		}
	}
	return payload, false
}
