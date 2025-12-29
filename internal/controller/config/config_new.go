package config

import (
	"gaap-api/api/config"
)

type ControllerV1 struct{}

func NewV1() config.IConfigV1 {
	return &ControllerV1{}
}
