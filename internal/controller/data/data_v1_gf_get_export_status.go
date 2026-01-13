package data

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/data/v1"
	taskV1 "gaap-api/api/task/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
)

func (c *ControllerV1) GfGetExportStatus(ctx context.Context, req *v1.GfGetExportStatusReq) (res *v1.GfGetExportStatusRes, err error) {
	taskId, err := uuid.Parse(req.GetTaskId())
	if err != nil {
		return nil, err
	}

	task, err := service.Data().GetExportStatus(ctx, taskId)
	if err != nil {
		return nil, err
	}

	return &v1.GetExportStatusRes{
		Task: modelTaskToProto(task),
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}

// modelTaskToProto converts model.Task to protobuf taskV1.Task
func modelTaskToProto(t *model.Task) *taskV1.Task {
	if t == nil {
		return nil
	}

	task := &taskV1.Task{
		Id:             t.Id.String(),
		Type:           taskV1.TaskType(t.Type),
		Status:         taskV1.TaskStatus(t.Status),
		Progress:       int32(t.Progress),
		TotalItems:     int32(t.TotalItems),
		ProcessedItems: int32(t.ProcessedItems),
	}

	// Convert payload and result to structpb.Struct if possible
	if t.Payload != nil {
		if payloadStruct, err := structpb.NewStruct(nil); err == nil {
			task.Payload = payloadStruct
		}
	}
	if t.Result != nil {
		if resultStruct, err := structpb.NewStruct(nil); err == nil {
			task.Result = resultStruct
		}
	}

	return task
}
