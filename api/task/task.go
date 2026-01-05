// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package task

import (
	"context"

	"gaap-api/api/task/v1"
)

type ITaskV1 interface {
	ListTasks(ctx context.Context, req *v1.ListTasksReq) (res *v1.ListTasksRes, err error)
	GetTask(ctx context.Context, req *v1.GetTaskReq) (res *v1.GetTaskRes, err error)
	CancelTask(ctx context.Context, req *v1.CancelTaskReq) (res *v1.CancelTaskRes, err error)
	RetryTask(ctx context.Context, req *v1.RetryTaskReq) (res *v1.RetryTaskRes, err error)
}
