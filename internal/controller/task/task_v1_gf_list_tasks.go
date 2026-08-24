package task

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/task/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfListTasks(ctx context.Context, req *v1.GfListTasksReq) (res *v1.GfListTasksRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.ListTasksReq); err != nil {
		return nil, err
	}

	input := model.TaskQueryInput{
		Page:   int(req.Query.Page),
		Limit:  int(req.Query.Limit),
		Status: int(req.Query.Status),
		Type:   int(req.Query.Type),
	}

	// Set defaults
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}

	tasks, total, err := service.Task().ListTasks(ctx, input)
	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int32(0)
	if input.Limit > 0 {
		totalPages = int32((total + input.Limit - 1) / input.Limit)
	}

	return &v1.ListTasksRes{
		Data: tasksToProtos(tasks),
		Pagination: &base.PaginatedResponse{
			Total:      int32(total),
			Page:       int32(input.Page),
			Limit:      int32(input.Limit),
			TotalPages: totalPages,
		},
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
