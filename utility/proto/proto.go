// Package proto provides utilities for handling Protobuf messages in ALE context.
package proto

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/protobuf/proto"
)

// ALEProtoBodyKey is the context key for storing decrypted protobuf body bytes.
const ALEProtoBodyKey = "ale_proto_body"

var (
	// ErrNoProtoBody indicates that no protobuf body was found in the context.
	ErrNoProtoBody = errors.New("no protobuf body found in context")
)

// formatProtoAsKeyValue converts a protobuf message to a readable key-value string
func formatProtoAsKeyValue(msg proto.Message) string {
	if msg == nil {
		return "{}"
	}

	// Get all fields from the message using reflection
	v := reflect.ValueOf(msg).Elem()
	t := v.Type()

	result := make(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Get json tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		// Handle omitempty and comma-separated tags
		key := jsonTag
		if idx := len(key); idx > 0 {
			if commaIdx := findChar(key, ','); commaIdx > 0 {
				key = key[:commaIdx]
			}
		}
		if key == "" {
			continue
		}

		// Get value
		value := fieldValue.Interface()
		if value == nil {
			continue
		}

		// Handle pointer types
		if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() {
			value = fieldValue.Elem().Interface()
		}

		result[key] = value
	}

	// Convert to JSON for readable output
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func findChar(s string, c rune) int {
	for i, ch := range s {
		if ch == c {
			return i
		}
	}
	return -1
}

// ParseFromALE extracts protobuf bytes from the ALE context and unmarshals into the target message.
// This function should be called at the beginning of controllers that receive ALE-encrypted requests.
//
// Example usage:
//
//	func (c *ControllerV1) GfRegister(ctx context.Context, req *v1.GfRegisterReq) (res *v1.GfRegisterRes, err error) {
//	    if err := proto.ParseFromALE(ctx, &req.RegisterReq); err != nil {
//	        return nil, err
//	    }
//	    // Now req fields are populated...
//	}
func ParseFromALE(ctx context.Context, target proto.Message) error {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return ErrNoProtoBody
	}

	protoBody := r.GetCtxVar(ALEProtoBodyKey)
	if protoBody.IsNil() || protoBody.IsEmpty() {
		// No ALE proto body in context, might be a non-ALE request
		// Return nil to allow fallback to regular parameter binding
		return nil
	}

	bytes := protoBody.Bytes()
	if len(bytes) == 0 {
		return nil
	}

	// 增加入参解析日志
	g.Log().Infof(ctx, "Parsing ALE protobuf request, target type: %T, raw bytes length: %d", target, len(bytes))
	g.Log().Debugf(ctx, "Raw bytes: %s", string(bytes))

	err := proto.Unmarshal(bytes, target)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to unmarshal ALE protobuf request: %v", err)
		return err
	}

	g.Log().Infof(ctx, "ALE Request: %s", formatProtoAsKeyValue(target))
	return nil
}

// HasALEProtoBody checks if the context contains an ALE protobuf body.
func HasALEProtoBody(ctx context.Context) bool {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return false
	}

	protoBody := r.GetCtxVar(ALEProtoBodyKey)
	return !protoBody.IsNil() && !protoBody.IsEmpty()
}
