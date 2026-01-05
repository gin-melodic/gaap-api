package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ExportParams defines parameters for data export
type ExportParams struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// ExportDataReq requests a data export task
type ExportDataReq struct {
	g.Meta `path:"/v1/data/export" tags:"Data" method:"post" summary:"Create data export task"`
	ExportParams
}

// ExportDataRes returns the created export task
type ExportDataRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	TaskId string `json:"taskId" dc:"Export task ID"`
}

// ImportDataReq requests a data import task
type ImportDataReq struct {
	g.Meta `path:"/v1/data/import" tags:"Data" method:"post" mime:"multipart/form-data" summary:"Create data import task"`
	File   *ghttp.UploadFile `json:"file" type:"file" v:"required" dc:"Export file to import (.zip)"`
}

// ImportDataRes returns the created import task
type ImportDataRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	TaskId string `json:"taskId" dc:"Import task ID"`
}

// DownloadExportReq requests download of completed export
type DownloadExportReq struct {
	g.Meta `path:"/v1/data/download/{taskId}" tags:"Data" method:"get" summary:"Download export file"`
	TaskId string `json:"taskId" in:"path" v:"required" dc:"Export task ID"`
}

// DownloadExportRes streams the export file
type DownloadExportRes struct {
	g.Meta `mime:"application/octet-stream"`
}

// GetExportStatusReq gets export task status
type GetExportStatusReq struct {
	g.Meta `path:"/v1/data/export/{taskId}" tags:"Data" method:"get" summary:"Get export task status"`
	TaskId string `json:"taskId" in:"path" v:"required" dc:"Export task ID"`
}

// GetExportStatusRes returns export task details
type GetExportStatusRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
	*common.Task[ExportParams, interface{}]
}
