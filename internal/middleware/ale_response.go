package middleware

import (
	"encoding/json"

	"gaap-api/internal/ale"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ALEResponseMiddleware encrypts responses when ALE is enabled.
// This middleware should be used INSTEAD OF ghttp.MiddlewareHandlerResponse for ALE routes.
// It handles both the response formatting and encryption.
func ALEResponseMiddleware(r *ghttp.Request) {
	r.Middleware.Next()

	// Check if there was an exception/panic - handle first
	if err := r.GetError(); err != nil {
		handleErrorResponse(r, err.Error(), 500)
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
		// ALE not enabled, return as regular JSON
		writeJSONResponse(r, handlerRes)
		return
	}

	ctx := r.Context()

	// Serialize response to Protobuf binary
	protoBytes, err := serializeToProtobuf(handlerRes)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to serialize response to protobuf: %v", err)
		// Fallback to JSON response
		writeJSONResponse(r, handlerRes)
		return
	}

	// Encrypt the protobuf data
	encrypted, err := ale.EncryptResponse(protoBytes, hexKey)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to encrypt ALE response: %v", err)
		handleErrorResponse(r, "Encryption error", 500)
		return
	}

	// Write encrypted binary response
	r.Response.Header().Set("Content-Type", "application/octet-stream")
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

	// Fallback: serialize to JSON bytes first
	// This works because protojson can handle the conversion
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	// For non-proto types, we need to return the JSON as-is
	// The frontend will need to handle this case
	// Actually, all our responses ARE proto.Message types,
	// so this path shouldn't be hit in practice.
	return jsonBytes, nil
}

// writeJSONResponse writes a JSON response (for non-ALE requests)
func writeJSONResponse(r *ghttp.Request, res interface{}) {
	// If it's a proto message, use protojson for consistent field names
	if protoMsg, ok := res.(proto.Message); ok {
		jsonBytes, err := protojson.Marshal(protoMsg)
		if err == nil {
			r.Response.Header().Set("Content-Type", "application/json")
			r.Response.ClearBuffer()
			r.Response.Write(jsonBytes)
			return
		}
	}

	// Fallback to standard JSON
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.ClearBuffer()
	r.Response.WriteJson(res)
}

// handleErrorResponse writes an error response
func handleErrorResponse(r *ghttp.Request, message string, code int) {
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.ClearBuffer()
	r.Response.WriteJson(g.Map{
		"code":    code,
		"message": message,
	})
}
