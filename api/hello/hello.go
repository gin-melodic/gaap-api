// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package hello

import (
	"context"

	"gaap-api/api/hello/v1"
)

type IHelloV1 interface {
	GfHello(ctx context.Context, req *v1.GfHelloReq) (res *v1.GfHelloRes, err error)
}
