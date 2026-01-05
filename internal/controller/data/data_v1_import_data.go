package data

import (
	"context"

	v1 "gaap-api/api/data/v1"
	dataLogic "gaap-api/internal/logic/data"
)

func (c *ControllerV1) ImportData(ctx context.Context, req *v1.ImportDataReq) (res *v1.ImportDataRes, err error) {
	return dataLogic.Data().Import(ctx, req)
}
