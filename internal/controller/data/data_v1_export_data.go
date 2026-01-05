package data

import (
	"context"

	v1 "gaap-api/api/data/v1"
	dataLogic "gaap-api/internal/logic/data"
)

func (c *ControllerV1) ExportData(ctx context.Context, req *v1.ExportDataReq) (res *v1.ExportDataRes, err error) {
	return dataLogic.Data().Export(ctx, req)
}
