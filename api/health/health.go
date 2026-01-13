// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package health

import (
	"context"

	"gaap-api/api/health/v1"
)

type IHealthV1 interface {
	GfHealth(ctx context.Context, req *v1.GfHealthReq) (res *v1.GfHealthRes, err error)
}
