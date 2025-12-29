package service

import (
	"context"

	"gaap-api/internal/model"
)

type ITask interface {
	// ListTasks returns a list of tasks for the current user
	ListTasks(ctx context.Context, in model.TaskQueryInput) (out []model.Task, total int, err error)
	// GetTask returns a single task by ID
	GetTask(ctx context.Context, id string) (out *model.Task, err error)
	// CreateTask creates a new task and publishes it to the queue
	CreateTask(ctx context.Context, in model.TaskCreateInput) (out *model.Task, err error)
	// CancelTask cancels a pending or running task
	CancelTask(ctx context.Context, id string) error
	// RetryTask retries a failed task
	RetryTask(ctx context.Context, id string) (*model.Task, error)
	// UpdateTaskProgress updates task progress
	UpdateTaskProgress(ctx context.Context, id string, progress int, processedItems int) error
	// CompleteTask marks a task as completed
	CompleteTask(ctx context.Context, id string, result interface{}) error
	// FailTask marks a task as failed
	FailTask(ctx context.Context, id string, errMsg string) error
	// StartWorker starts the background task worker
	StartWorker(ctx context.Context) error
}

var localTask ITask

func Task() ITask {
	if localTask == nil {
		panic("implement not found for interface ITask, forgot register?")
	}
	return localTask
}

func RegisterTask(i ITask) {
	localTask = i
}
