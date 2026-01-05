// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package debug

import (
	"context"

	"gaap-api/api/debug/v1"
)

type IDebugV1 interface {
	ExecSql(ctx context.Context, req *v1.ExecSqlReq) (res *v1.ExecSqlRes, err error)
}
