package middleware

import (
	"fmt"
	"strings"

	"gaap-api/internal/ale"

	"github.com/gogf/gf/v2/net/ghttp"
	"google.golang.org/protobuf/proto"
)

// ALEResponseMiddleware encrypts responses when ALE is enabled.
// This middleware should be used INSTEAD OF ghttp.MiddlewareHandlerResponse for ALE routes.
// It handles both the response formatting and encryption.
func ALEResponseMiddleware(r *ghttp.Request) {
	r.Middleware.Next()

	// Check if there was an exception/panic - handle first
	if err := r.GetError(); err != nil {
		status, message := classifyHandlerError(err.Error())
		writeProtoError(r, status, message, r.GetCtxVar("ale_key").String())
		return
	}

	// Get handler response
	handlerRes := r.GetHandlerResponse()
	if handlerRes == nil {
		// No response, skip encryption
		return
	}

	// Check if ALE is enabled for this request
	aleEnabled := r.GetCtxVar("ale_enabled")
	hexKey := r.GetCtxVar("ale_key").String()

	if aleEnabled.IsNil() || !aleEnabled.Bool() || hexKey == "" {
		writeProtoError(r, 500, "secure response unavailable", "")
		return
	}

	// Serialize response to Protobuf binary
	protoBytes, err := serializeToProtobuf(handlerRes)
	if err != nil {
		writeProtoError(r, 500, "invalid protobuf response", hexKey)
		return
	}

	// Encrypt the protobuf data
	encrypted, err := ale.EncryptResponse(protoBytes, hexKey)
	if err != nil {
		writeProtoError(r, 500, "response encryption failed", "")
		return
	}

	// Write encrypted binary response
	r.Response.Header().Set("Content-Type", "application/octet-stream")
	r.Response.Header().Set(HeaderALEEncrypted, "1")
	r.Response.ClearBuffer()
	r.Response.Write(encrypted)
}

// serializeToProtobuf attempts to serialize the response to protobuf binary.
// If the response is a proto.Message, use proto.Marshal.
// Otherwise, convert to JSON first, then to protobuf.
func serializeToProtobuf(res interface{}) ([]byte, error) {
	// If it's already a proto.Message, serialize directly
	if protoMsg, ok := res.(proto.Message); ok {
		return proto.Marshal(protoMsg)
	}

	return nil, fmt.Errorf("response type %T is not a protobuf message", res)
}

func classifyHandlerError(raw string) (int, string) {
	message := strings.TrimSpace(raw)
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "invalid email or password"):
		return 401, message
	case strings.Contains(lower, "registration unavailable"):
		return 403, message
	case strings.Contains(lower, "not found"):
		return 404, message
	case strings.Contains(lower, "unauthorized"), strings.Contains(lower, "token"), strings.Contains(lower, "session expired"):
		return 401, message
	case strings.Contains(lower, "required"), strings.Contains(lower, "invalid"), strings.Contains(lower, "cannot"),
		strings.Contains(lower, "mismatch"), strings.Contains(lower, "unknown"), strings.Contains(lower, "amount"),
		strings.Contains(lower, "currency"), strings.Contains(lower, "account"), strings.Contains(lower, "email"),
		strings.Contains(lower, "password"):
		return 400, message
	default:
		return 500, "internal server error"
	}
}
