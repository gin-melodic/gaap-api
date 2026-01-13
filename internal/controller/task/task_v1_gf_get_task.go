package task

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/task/v1"
	"gaap-api/internal/service"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfGetTask(ctx context.Context, req *v1.GfGetTaskReq) (res *v1.GfGetTaskRes, err error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	task, err := service.Task().GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.GetTaskRes{
		Task: taskToProto(task),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
