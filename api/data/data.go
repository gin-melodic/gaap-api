// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package data

import (
	"context"

	"gaap-api/api/data/v1"
)

type IDataV1 interface {
	GfExportData(ctx context.Context, req *v1.GfExportDataReq) (res *v1.GfExportDataRes, err error)
	GfImportData(ctx context.Context, req *v1.GfImportDataReq) (res *v1.GfImportDataRes, err error)
	GfDownloadExport(ctx context.Context, req *v1.GfDownloadExportReq) (res *v1.GfDownloadExportRes, err error)
	GfGetExportStatus(ctx context.Context, req *v1.GfGetExportStatusReq) (res *v1.GfGetExportStatusRes, err error)
}
