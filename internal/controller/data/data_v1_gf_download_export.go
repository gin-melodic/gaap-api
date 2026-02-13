package data

import (
	"context"

	v1 "gaap-api/api/data/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

func (c *ControllerV1) GfDownloadExport(ctx context.Context, req *v1.GfDownloadExportReq) (res *v1.GfDownloadExportRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.DownloadExportReq); err != nil {
		return nil, err
	}

	taskId, err := uuid.Parse(req.GetTaskId())
	if err != nil {
		return nil, err
	}

	input := model.DataDownloadInput{
		TaskId: taskId,
	}

	// Get the HTTP request to serve the file
	r := ghttp.RequestFromCtx(ctx)

	err = service.Data().Download(ctx, input, r)
	if err != nil {
		return nil, err
	}

	// Response is handled by the service (file download)
	return nil, nil
}
