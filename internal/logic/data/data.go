package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gaap-api/internal/dataimport"
	"gaap-api/internal/export"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// sData is the service implementation for data export/import
type sData struct{}

var dataInstance = &sData{}

func init() {
	service.RegisterData(New())
}

// New returns the data service instance
func New() *sData {
	return dataInstance
}

// Export creates an export task
func (s *sData) Export(ctx context.Context, in model.DataExportInput) (*model.DataExportOutput, error) {
	userIdStr := utils.RequireUserId(ctx)
	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return nil, gerror.Wrap(err, "invalid user ID")
	}

	// Validate date range
	startDate, err := time.Parse("2006-01-02", in.StartDate)
	if err != nil {
		return nil, gerror.New("invalid start date format")
	}

	endDate, err := time.Parse("2006-01-02", in.EndDate)
	if err != nil {
		return nil, gerror.New("invalid end date format")
	}

	if endDate.Before(startDate) {
		return nil, gerror.New("end date must be after start date")
	}

	// Check max 3 years
	maxEnd := startDate.AddDate(3, 0, 0)
	if endDate.After(maxEnd) {
		return nil, gerror.New("date range cannot exceed 3 years")
	}

	// Create export task
	task, err := service.Task().CreateTask(ctx, model.TaskCreateInput[any]{
		UserId: userId,
		Type:   model.TaskTypeDataExport,
		Payload: model.DataExportPayload{
			Payload:   &model.Payload{UserId: userId},
			StartDate: in.StartDate,
			EndDate:   in.EndDate,
		},
	})
	if err != nil {
		return nil, err
	}

	return &model.DataExportOutput{
		TaskId: task.Id.String(),
	}, nil
}

// Import creates an import task
func (s *sData) Import(ctx context.Context, in model.DataImportInput) (*model.DataImportOutput, error) {
	userIdStr := utils.RequireUserId(ctx)
	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return nil, gerror.Wrap(err, "invalid user ID")
	}

	// Check if user already has an active import task
	hasActive, err := dataimport.HasActiveImportTask(ctx, userIdStr)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to check import status")
	}
	if hasActive {
		return nil, gerror.New("an import task is already in progress")
	}

	// Save uploaded file
	if len(in.FileContent) == 0 {
		return nil, gerror.New("no file uploaded")
	}

	// Validate file extension
	if filepath.Ext(in.FileName) != ".zip" {
		return nil, gerror.New("only .zip files are supported")
	}

	// Generate unique filename
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("import_%s_%s_%s", userIdStr[:8], timestamp, in.FileName)

	// Save to import directory
	if err := os.MkdirAll(dataimport.ImportDir, 0755); err != nil {
		return nil, gerror.Wrap(err, "failed to create import directory")
	}

	filePath := filepath.Join(dataimport.ImportDir, fileName)
	if err := os.WriteFile(filePath, in.FileContent, 0644); err != nil {
		return nil, gerror.Wrap(err, "failed to save file")
	}

	// Create import task
	task, err := service.Task().CreateTask(ctx, model.TaskCreateInput[any]{
		UserId: userId,
		Type:   model.TaskTypeDataImport,
		Payload: model.DataImportPayload{
			Payload:  &model.Payload{UserId: userId},
			FileName: filePath,
		},
	})
	if err != nil {
		// Clean up file on error
		os.Remove(filePath)
		return nil, err
	}

	return &model.DataImportOutput{
		TaskId: task.Id.String(),
	}, nil
}

// Download serves the export file for download
func (s *sData) Download(ctx context.Context, in model.DataDownloadInput, r *ghttp.Request) error {
	userIdStr := utils.RequireUserId(ctx)
	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return gerror.Wrap(err, "invalid user ID")
	}

	// Get task
	task, err := service.Task().GetTask(ctx, in.TaskId)
	if err != nil {
		return err
	}

	// Verify ownership
	if task.UserId != userId {
		return gerror.New("task not found")
	}

	// Check task status
	if task.Status != model.TaskStatusCompleted {
		return gerror.New("export not ready")
	}

	// Get result
	result, ok := task.Result.(map[string]interface{})
	if !ok {
		return gerror.New("invalid task result")
	}

	filePath, _ := result["filePath"].(string)
	fileName, _ := result["fileName"].(string)

	if filePath == "" || fileName == "" {
		return gerror.New("export file not found")
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return gerror.New("export file has expired")
	}

	// Serve file
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	r.Response.Header().Set("Content-Type", "application/zip")
	r.Response.ServeFile(filePath)

	// Clean up after download
	go export.CleanupExport(filePath)

	return nil
}

// GetExportStatus returns the status of an export task
func (s *sData) GetExportStatus(ctx context.Context, taskId uuid.UUID) (*model.TaskOutput[any, any], error) {
	userIdStr := utils.RequireUserId(ctx)
	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return nil, gerror.Wrap(err, "invalid user ID")
	}

	task, err := service.Task().GetTask(ctx, taskId)
	if err != nil {
		return nil, err
	}

	if task.UserId != userId {
		return nil, gerror.New("task not found")
	}

	return task, nil
}

// CheckImportLock checks if user has an active import that blocks mutations
func CheckImportLock(ctx context.Context) error {
	userId := utils.RequireUserId(ctx)

	hasActive, err := dataimport.HasActiveImportTask(ctx, userId)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to check import lock: %v", err)
		return nil // Don't block on check failure
	}

	if hasActive {
		return gerror.New("操作已暂停：正在导入数据，请等待导入完成后再试")
	}

	return nil
}
