// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"gaap-api/internal/model"
)

type (
	IConsole interface {
		// Status assembles the admin-console observation payload: dependency
		// health, ALE/auth counters, HTTP stats, reconciliation and user activity.
		Status(ctx context.Context) (out *model.OpsStatus, err error)
		// ReconcileNow runs the read-only ledger reconciler on demand.
		ReconcileNow(ctx context.Context) (out *model.Report, err error)
	}
)

var (
	localConsole IConsole
)

func Console() IConsole {
	if localConsole == nil {
		panic("implement not found for interface IConsole, forgot register?")
	}
	return localConsole
}

func RegisterConsole(i IConsole) {
	localConsole = i
}
