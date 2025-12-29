package health

import (
	"gaap-api/api/health"
)

type ControllerV1 struct{}

func NewV1() health.IHealthV1 {
	return &ControllerV1{}
}
