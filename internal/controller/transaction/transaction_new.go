package transaction

import (
	"gaap-api/api/transaction"
)

type ControllerV1 struct{}

func NewV1() transaction.ITransactionV1 {
	return &ControllerV1{}
}
