package debug

import (
	"context"
	v1 "gaap-api/api/debug/v1"

	"github.com/gogf/gf/v2/frame/g"
)

func (c *ControllerV1) ExecSql(ctx context.Context, req *v1.ExecSqlReq) (res *v1.ExecSqlRes, err error) {
	// Execute the raw SQL using g.DB().Ctx(ctx).GetAll(ctx, req.Sql)
	// We use GetAll to retrieve results as []Record

	result, err := g.DB().Ctx(ctx).GetAll(ctx, req.Sql)
	if err != nil {
		return nil, err
	}

	// Convert result (type Result = []Record) to []map[string]interface{}
	listMap := result.List()

	res = &v1.ExecSqlRes{
		Result: listMap,
	}
	return
}
