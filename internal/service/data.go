// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type (
	IData interface {
		// Export creates an export task
		Export(ctx context.Context, in model.DataExportInput) (*model.DataExportOutput, error)
		// Import creates an import task
		Import(ctx context.Context, in model.DataImportInput) (*model.DataImportOutput, error)
		// Download serves the export file for download
		Download(ctx context.Context, in model.DataDownloadInput, r *ghttp.Request) error
		// GetExportStatus returns the status of an export task
		GetExportStatus(ctx context.Context, taskId uuid.UUID) (*model.TaskOutput[any, any], error)
	}
)

var (
	localData IData
)

func Data() IData {
	if localData == nil {
		panic("implement not found for interface IData, forgot register?")
	}
	return localData
}

func RegisterData(i IData) {
	localData = i
}
