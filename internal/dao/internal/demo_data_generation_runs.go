// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

type DemoDataGenerationRunsDao struct {
	table    string
	group    string
	columns  DemoDataGenerationRunsColumns
	handlers []gdb.ModelHandler
}

type DemoDataGenerationRunsColumns struct {
	UserId         string
	BusinessDate   string
	GeneratedCount string
	CreatedAt      string
}

var demoDataGenerationRunsColumns = DemoDataGenerationRunsColumns{
	UserId:         "user_id",
	BusinessDate:   "business_date",
	GeneratedCount: "generated_count",
	CreatedAt:      "created_at",
}

func NewDemoDataGenerationRunsDao(handlers ...gdb.ModelHandler) *DemoDataGenerationRunsDao {
	return &DemoDataGenerationRunsDao{
		group:    "default",
		table:    "demo_data_generation_runs",
		columns:  demoDataGenerationRunsColumns,
		handlers: handlers,
	}
}

func (dao *DemoDataGenerationRunsDao) DB() gdb.DB                             { return g.DB(dao.group) }
func (dao *DemoDataGenerationRunsDao) Table() string                          { return dao.table }
func (dao *DemoDataGenerationRunsDao) Columns() DemoDataGenerationRunsColumns { return dao.columns }
func (dao *DemoDataGenerationRunsDao) Group() string                          { return dao.group }

func (dao *DemoDataGenerationRunsDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

func (dao *DemoDataGenerationRunsDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) error {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
