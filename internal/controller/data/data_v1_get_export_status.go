package data

import (
	"context"

	v1 "gaap-api/api/data/v1"
	dataLogic "gaap-api/internal/logic/data"
)

func (c *ControllerV1) GetExportStatus(ctx context.Context, req *v1.GetExportStatusReq) (res *v1.GetExportStatusRes, err error) {
	return dataLogic.Data().GetExportStatus(ctx, req)
}
