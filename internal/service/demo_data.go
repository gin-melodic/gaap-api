// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
)

type (
	IDemoData interface {
		// StartScheduler starts the demo catch-up loop when explicitly enabled.
		StartScheduler(ctx context.Context) error
		// CatchUp generates every unfinished business date from the configured start
		// date through yesterday in the configured timezone.
		CatchUp(ctx context.Context) (int, error)
	}
)

var (
	localDemoData IDemoData
)

func DemoData() IDemoData {
	if localDemoData == nil {
		panic("implement not found for interface IDemoData, forgot register?")
	}
	return localDemoData
}

func RegisterDemoData(i IDemoData) {
	localDemoData = i
}
