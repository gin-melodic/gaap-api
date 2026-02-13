// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"

	"github.com/google/uuid"
)

type (
	ITask interface {
		// ListTasks returns a list of tasks for the current user
		ListTasks(ctx context.Context, in model.TaskQueryInput) (out []model.TaskOutput[any, any], total int, err error)
		// GetTask returns a single task by ID
		GetTask(ctx context.Context, id uuid.UUID) (out *model.TaskOutput[any, any], err error)
		// CreateTask creates a new task and publishes it to the queue
		CreateTask(ctx context.Context, in model.TaskCreateInput[any]) (out *model.TaskOutput[any, any], err error)
		// CancelTask cancels a pending or running task
		CancelTask(ctx context.Context, id uuid.UUID) error
		// RetryTask retries a failed task
		RetryTask(ctx context.Context, id uuid.UUID) (*model.TaskOutput[any, any], error)
		// UpdateTaskProgress updates task progress
		UpdateTaskProgress(ctx context.Context, id uuid.UUID, progress int, processedItems int) error
		// CompleteTask marks a task as completed
		CompleteTask(ctx context.Context, id uuid.UUID, result interface{}) error
		// FailTask marks a task as failed
		FailTask(ctx context.Context, id uuid.UUID, errMsg string) error
		// StartWorker starts the background task worker
		StartWorker(ctx context.Context) error
	}
)

var (
	localTask ITask
)

func Task() ITask {
	if localTask == nil {
		panic("implement not found for interface ITask, forgot register?")
	}
	return localTask
}

func RegisterTask(i ITask) {
	localTask = i
}
