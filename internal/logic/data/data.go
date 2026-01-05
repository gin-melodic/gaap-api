package data

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	common "gaap-api/api/common/v1"
	v1 "gaap-api/api/data/v1"
	"gaap-api/internal/dataimport"
	"gaap-api/internal/export"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// sData is the service implementation for data export/import
type sData struct{}

var dataInstance = &sData{}

// Data returns the data service instance
func Data() *sData {
	return dataInstance
}

// Export creates an export task
func (s *sData) Export(ctx context.Context, req *v1.ExportDataReq) (*v1.ExportDataRes, error) {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	if userId == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Validate date range
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format")
	}

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Check max 3 years
	maxEnd := startDate.AddDate(3, 0, 0)
	if endDate.After(maxEnd) {
		return nil, fmt.Errorf("date range cannot exceed 3 years")
	}

	// Create export task
	task, err := service.Task().CreateTask(ctx, model.TaskCreateInput{
		UserId: userId,
		Type:   model.TaskTypeDataExport,
		Payload: model.DataExportPayload{
			UserId:    userId,
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
		},
	})
	if err != nil {
		return nil, err
	}

	return &v1.ExportDataRes{
		TaskId: task.Id,
	}, nil
}

// Import creates an import task
func (s *sData) Import(ctx context.Context, req *v1.ImportDataReq) (*v1.ImportDataRes, error) {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	if userId == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Check if user already has an active import task
	hasActive, err := dataimport.HasActiveImportTask(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to check import status: %w", err)
	}
	if hasActive {
		return nil, fmt.Errorf("an import task is already in progress")
	}

	// Save uploaded file
	file := req.File
	if file == nil {
		return nil, fmt.Errorf("no file uploaded")
	}

	// Validate file extension
	if filepath.Ext(file.Filename) != ".zip" {
		return nil, fmt.Errorf("only .zip files are supported")
	}

	// Generate unique filename
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("import_%s_%s_%s", userId[:8], timestamp, file.Filename)

	// Save to import directory
	if err := os.MkdirAll(dataimport.ImportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create import directory: %w", err)
	}

	filePath := filepath.Join(dataimport.ImportDir, fileName)
	savedFile, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer savedFile.Close()

	destFile, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, savedFile); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create import task
	task, err := service.Task().CreateTask(ctx, model.TaskCreateInput{
		UserId: userId,
		Type:   model.TaskTypeDataImport,
		Payload: model.DataImportPayload{
			UserId:   userId,
			FileName: filePath,
		},
	})
	if err != nil {
		// Clean up file on error
		os.Remove(filePath)
		return nil, err
	}

	return &v1.ImportDataRes{
		TaskId: task.Id,
	}, nil
}

// Download serves the export file for download
func (s *sData) Download(ctx context.Context, req *v1.DownloadExportReq, r *ghttp.Request) error {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	if userId == "" {
		return fmt.Errorf("user not authenticated")
	}

	// Get task
	task, err := service.Task().GetTask(ctx, req.TaskId)
	if err != nil {
		return err
	}

	// Verify ownership
	if task.UserId != userId {
		return fmt.Errorf("task not found")
	}

	// Check task status
	if task.Status != model.TaskStatusCompleted {
		return fmt.Errorf("export not ready")
	}

	// Get result
	result, ok := task.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid task result")
	}

	filePath, _ := result["filePath"].(string)
	fileName, _ := result["fileName"].(string)

	if filePath == "" || fileName == "" {
		return fmt.Errorf("export file not found")
	}

	// Check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("export file has expired")
	}

	// Serve file
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	r.Response.Header().Set("Content-Type", "application/zip")
	r.Response.ServeFile(filePath)

	// Clean up after download (optional - could also use scheduled cleanup)
	go export.CleanupExport(filePath)

	return nil
}

// GetExportStatus returns the status of an export task
func (s *sData) GetExportStatus(ctx context.Context, req *v1.GetExportStatusReq) (*v1.GetExportStatusRes, error) {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	if userId == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	task, err := service.Task().GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	if task.UserId != userId {
		return nil, fmt.Errorf("task not found")
	}

	// Parse payload
	var payload model.DataExportPayload
	if task.Payload != nil {
		if p, ok := task.Payload.(map[string]interface{}); ok {
			// If payload is a map (from JSON unmarshal), convert it locally or just extract fields
			// Since model.Task.Payload is interface{}, it might be a map or the struct depending on how it was loaded
			// Here we assume standard JSON unmarshaling into interface{} resulted in a map
			startDate, _ := p["startDate"].(string)
			endDate, _ := p["endDate"].(string)
			payload.StartDate = startDate
			payload.EndDate = endDate
		}
		// If using gdb scan into model.Task, it might have been unmarshaled into map[string]interface{}
	}

	return &v1.GetExportStatusRes{
		Task: &common.Task[v1.ExportParams, interface{}]{
			TaskId:   task.Id,
			Status:   task.Status,
			Progress: task.Progress,
			Payload: v1.ExportParams{
				StartDate: payload.StartDate,
				EndDate:   payload.EndDate,
			},
			Result: task.Result,
		},
	}, nil
}

// CheckImportLock checks if user has an active import that blocks mutations
func CheckImportLock(ctx context.Context) error {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	if userId == "" {
		return nil
	}

	hasActive, err := dataimport.HasActiveImportTask(ctx, userId)
	if err != nil {
		g.Log().Warningf(ctx, "Failed to check import lock: %v", err)
		return nil // Don't block on check failure
	}

	if hasActive {
		return fmt.Errorf("操作已暂停：正在导入数据，请等待导入完成后再试")
	}

	return nil
}
