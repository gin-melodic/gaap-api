// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package task

import (
	"context"

	"gaap-api/api/task/v1"
)

type ITaskV1 interface {
	GfListTasks(ctx context.Context, req *v1.GfListTasksReq) (res *v1.GfListTasksRes, err error)
	GfGetTask(ctx context.Context, req *v1.GfGetTaskReq) (res *v1.GfGetTaskRes, err error)
	GfCancelTask(ctx context.Context, req *v1.GfCancelTaskReq) (res *v1.GfCancelTaskRes, err error)
	GfRetryTask(ctx context.Context, req *v1.GfRetryTaskReq) (res *v1.GfRetryTaskRes, err error)
}
