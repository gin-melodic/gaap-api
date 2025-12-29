package dashboard

import (
	"gaap-api/api/dashboard"
)

type ControllerV1 struct{}

func NewV1() dashboard.IDashboardV1 {
	return &ControllerV1{}
}
