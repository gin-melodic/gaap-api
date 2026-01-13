package task

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/task/v1"
	"gaap-api/internal/service"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfRetryTask(ctx context.Context, req *v1.GfRetryTaskReq) (res *v1.GfRetryTaskRes, err error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	task, err := service.Task().RetryTask(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.RetryTaskRes{
		Task: taskToProto(task),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
