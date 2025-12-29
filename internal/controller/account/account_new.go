package account

import (
	"gaap-api/api/account"
)

type ControllerV1 struct{}

func NewV1() account.IAccountV1 {
	return &ControllerV1{}
}
