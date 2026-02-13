package health

import (
	"context"

	v1 "gaap-api/api/health/v1"
)

func (c *ControllerV1) GfHealth(ctx context.Context, req *v1.GfHealthReq) (res *v1.GfHealthRes, err error) {
	return &v1.HealthRes{
		Status: "healthy",
	}, nil
}
