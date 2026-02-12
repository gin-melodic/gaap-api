package task

import (
	"context"
	"encoding/json"

	"gaap-api/internal/dao"
	"gaap-api/internal/dataimport"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// processDataImport handles data import task
func (s *sTask) processDataImport(ctx context.Context, payload json.RawMessage) error {
	var data struct {
		TaskId  string                  `json:"taskId"`
		Payload model.DataImportPayload `json:"payload"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return gerror.Wrap(err, "failed to unmarshal import payload")
	}

	taskId, err := uuid.Parse(data.TaskId)
	if err != nil {
		return gerror.Wrap(err, "invalid task ID")
	}
	importPayload := data.Payload

	// Update task status to running
	_, err = dao.Tasks.Ctx(ctx).Where(dao.Tasks.Columns().Id, taskId).Data(entity.Tasks{
		Status:    model.TaskStatusRunning,
		StartedAt: gtime.Now(),
	}).Update()
	if err != nil {
		return gerror.Wrap(err, "failed to update task status")
	}

	// Check if task was cancelled
	task, err := s.GetTask(ctx, taskId)
	if err != nil || task.Status == model.TaskStatusCancelled {
		return nil
	}

	// Execute import
	importResult, err := s.executeDataImport(ctx, taskId, &importPayload)
	if err != nil {
		s.FailTask(ctx, taskId, err.Error())
		return err
	}

	// Trigger dashboard snapshot rebuild after import (bulk transactions created)
	if importPayload.UserId != uuid.Nil {
		dashboard.PublishDashboardRefresh(ctx, importPayload.UserId.String(), "data_import")
	}

	return s.CompleteTask(ctx, taskId, importResult)
}

// executeDataImport performs the actual data import
func (s *sTask) executeDataImport(ctx context.Context, taskId uuid.UUID, payload *model.DataImportPayload) (*model.DataImportResult, error) {
	result, err := dataimport.ImportData(ctx, payload.UserId.String(), payload.FileName)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to import data")
	}

	return &model.DataImportResult{
		AccountsImported:     result.AccountsImported,
		TransactionsImported: result.TransactionsImported,
		AccountsSkipped:      result.AccountsSkipped,
		TransactionsSkipped:  result.TransactionsSkipped,
	}, nil
}
