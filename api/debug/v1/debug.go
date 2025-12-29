package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
)

type ExecSqlReq struct {
	g.Meta `path:"/debug/sql" tags:"Debug" method:"post" summary:"Execute raw SQL"`
	Sql    string `json:"sql" v:"required"`
}

type ExecSqlRes struct {
	*common.BaseResponse
	Result interface{} `json:"result,omitempty"`
}
