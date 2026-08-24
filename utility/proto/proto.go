// Package proto provides utilities for handling Protobuf messages in ALE context.
package proto

import (
	"context"
	"errors"

	"github.com/gogf/gf/v2/frame/g"
	"google.golang.org/protobuf/proto"
)

// ALEProtoBodyKey is the context key for storing decrypted protobuf body bytes.
const ALEProtoBodyKey = "ale_proto_body"

var (
	// ErrNoProtoBody indicates that no protobuf body was found in the context.
	ErrNoProtoBody = errors.New("no protobuf body found in context")
)

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

	err := proto.Unmarshal(bytes, target)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to unmarshal ALE protobuf request: %v", err)
		return err
	}

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
