package task

import (
	"context"
	"encoding/json"

	"gaap-api/internal/dao"
	exportPkg "gaap-api/internal/export"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// processDataExport handles data export task
func (s *sTask) processDataExport(ctx context.Context, payload json.RawMessage) error {
	var data struct {
		TaskId  string                  `json:"taskId"`
		Payload model.DataExportPayload `json:"payload"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return gerror.Wrap(err, "failed to unmarshal export payload")
	}

	taskId, err := uuid.Parse(data.TaskId)
	if err != nil {
		return gerror.Wrap(err, "invalid task ID")
	}
	exportPayload := data.Payload

	// Update task status to running
	_, err = dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, taskId).Data(g.Map{
		dao.Tasks.Columns().Status:    model.TaskStatusRunning,
		dao.Tasks.Columns().StartedAt: gtime.Now(),
	}).Update()
	if err != nil {
		return gerror.Wrap(err, "failed to update task status")
	}

	// Add userId to context for GetTask
	ctx = context.WithValue(ctx, middleware.UserIdKey, exportPayload.UserId.String())

	// Check if task was cancelled
	task, err := s.GetTask(ctx, taskId)
	if err != nil || task.Status == model.TaskStatusCancelled {
		return nil
	}

	// Execute export
	exportResult, err := s.executeDataExport(ctx, taskId, &exportPayload)
	if err != nil {
		s.FailTask(ctx, taskId, err.Error())
		return err
	}

	return s.CompleteTask(ctx, taskId, exportResult)
}

// executeDataExport performs the actual data export
func (s *sTask) executeDataExport(ctx context.Context, taskId uuid.UUID, payload *model.DataExportPayload) (*model.DataExportResult, error) {
	result, err := exportPkg.CreateExport(ctx, payload.UserId.String(), payload.StartDate, payload.EndDate)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to create export")
	}

	return &model.DataExportResult{
		FilePath:             result.FilePath,
		FileName:             result.FileName,
		FileSize:             result.FileSize,
		AccountsExported:     result.AccountsExported,
		TransactionsExported: result.TransactionsExported,
	}, nil
}
