package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

type HealthReq struct {
	g.Meta `path:"/health" tags:"Health" method:"get" summary:"Health check"`
}

type HealthRes struct {
	g.Meta `mime:"application/json"`
	Status string `json:"status"`
}
