package health

import (
	"context"

	v1 "gaap-api/api/health/v1"
)

type IHealthV1 interface {
	Health(ctx context.Context, req *v1.HealthReq) (res *v1.HealthRes, err error)
}
