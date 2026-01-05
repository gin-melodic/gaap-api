// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package data

import (
	"context"

	"gaap-api/api/data/v1"
)

type IDataV1 interface {
	ExportData(ctx context.Context, req *v1.ExportDataReq) (res *v1.ExportDataRes, err error)
	ImportData(ctx context.Context, req *v1.ImportDataReq) (res *v1.ImportDataRes, err error)
	DownloadExport(ctx context.Context, req *v1.DownloadExportReq) (res *v1.DownloadExportRes, err error)
	GetExportStatus(ctx context.Context, req *v1.GetExportStatusReq) (res *v1.GetExportStatusRes, err error)
}
