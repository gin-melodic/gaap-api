package task

import (
	"context"
	v1 "gaap-api/api/task/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
)

type ControllerV1 struct{}

func NewV1() *ControllerV1 {
	return &ControllerV1{}
}

// TaskList handles GET /tasks
func (c *ControllerV1) TaskList(ctx context.Context, req *v1.ListTasksReq) (res *v1.ListTasksRes, err error) {
	tasks, total, err := service.Task().ListTasks(ctx, model.TaskQueryInput{
		Page:   req.Page,
		Limit:  req.Limit,
		Status: req.Status,
		Type:   req.Type,
	})
	if err != nil {
		return nil, err
	}

	res = &v1.ListTasksRes{
		Data: make([]v1.Task, len(tasks)),
	}
	res.Total = total
	res.Page = req.Page
	res.Limit = req.Limit

	for i, t := range tasks {
		res.Data[i] = v1.Task{
			Id:             t.Id,
			Type:           t.Type,
			Status:         t.Status,
			Payload:        t.Payload,
			Result:         t.Result,
			Progress:       t.Progress,
			TotalItems:     t.TotalItems,
			ProcessedItems: t.ProcessedItems,
			StartedAt:      t.StartedAt,
			CompletedAt:    t.CompletedAt,
			CreatedAt:      t.CreatedAt,
			UpdatedAt:      t.UpdatedAt,
		}
	}

	return
}

// TaskGet handles GET /tasks/{id}
func (c *ControllerV1) TaskGet(ctx context.Context, req *v1.GetTaskReq) (res *v1.GetTaskRes, err error) {
	task, err := service.Task().GetTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	res = &v1.GetTaskRes{
		Task: &v1.Task{
			Id:             task.Id,
			Type:           task.Type,
			Status:         task.Status,
			Payload:        task.Payload,
			Result:         task.Result,
			Progress:       task.Progress,
			TotalItems:     task.TotalItems,
			ProcessedItems: task.ProcessedItems,
			StartedAt:      task.StartedAt,
			CompletedAt:    task.CompletedAt,
			CreatedAt:      task.CreatedAt,
			UpdatedAt:      task.UpdatedAt,
		},
	}
	return
}

// TaskCancel handles POST /tasks/{id}/cancel
func (c *ControllerV1) TaskCancel(ctx context.Context, req *v1.CancelTaskReq) (res *v1.CancelTaskRes, err error) {
	err = service.Task().CancelTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	res = &v1.CancelTaskRes{}
	return
}

// TaskRetry handles POST /tasks/{id}/retry
func (c *ControllerV1) TaskRetry(ctx context.Context, req *v1.RetryTaskReq) (res *v1.RetryTaskRes, err error) {
	task, err := service.Task().RetryTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	res = &v1.RetryTaskRes{
		Task: &v1.Task{
			Id:             task.Id,
			Type:           task.Type,
			Status:         task.Status,
			Payload:        task.Payload,
			Result:         task.Result,
			Progress:       task.Progress,
			TotalItems:     task.TotalItems,
			ProcessedItems: task.ProcessedItems,
			StartedAt:      task.StartedAt,
			CompletedAt:    task.CompletedAt,
			CreatedAt:      task.CreatedAt,
			UpdatedAt:      task.UpdatedAt,
		},
	}
	return
}
