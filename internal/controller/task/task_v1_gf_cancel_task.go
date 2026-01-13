package task

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/task/v1"
	"gaap-api/internal/service"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfCancelTask(ctx context.Context, req *v1.GfCancelTaskReq) (res *v1.GfCancelTaskRes, err error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	err = service.Task().CancelTask(ctx, id)
	if err != nil {
		return nil, err
	}

	return &v1.CancelTaskRes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
