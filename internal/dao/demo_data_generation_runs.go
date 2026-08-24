package dao

import "gaap-api/internal/dao/internal"

type demoDataGenerationRunsDao struct {
	*internal.DemoDataGenerationRunsDao
}

var DemoDataGenerationRuns = demoDataGenerationRunsDao{internal.NewDemoDataGenerationRunsDao()}
