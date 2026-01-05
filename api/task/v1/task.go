package v1

import (
	common "gaap-api/api/common/v1"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type Task struct {
	Id             string      `json:"id"`
	Type           string      `json:"type"`
	Status         string      `json:"status"`
	Payload        interface{} `json:"payload"`
	Result         interface{} `json:"result,omitempty"`
	Progress       int         `json:"progress"`
	TotalItems     int         `json:"totalItems"`
	ProcessedItems int         `json:"processedItems"`
	StartedAt      *gtime.Time `json:"startedAt,omitempty"`
	CompletedAt    *gtime.Time `json:"completedAt,omitempty"`
	CreatedAt      *gtime.Time `json:"createdAt"`
	UpdatedAt      *gtime.Time `json:"updatedAt"`
}

type TaskQuery struct {
	Page   int    `json:"page" v:"min:1" d:"1"`
	Limit  int    `json:"limit" v:"min:1|max:100" d:"20"`
	Status string `json:"status" v:"in:PENDING,RUNNING,COMPLETED,FAILED,CANCELLED"`
	Type   string `json:"type"`
}

type ListTasksReq struct {
	g.Meta `path:"/v1/tasks" tags:"Tasks" method:"get" summary:"List all tasks for current user"`
	TaskQuery
}

type ListTasksRes struct {
	g.Meta `mime:"application/json"`
	common.PaginatedResponse
	*common.BaseResponse
	Data []Task `json:"data"`
}

type GetTaskReq struct {
	g.Meta `path:"/v1/tasks/{id}" tags:"Tasks" method:"get" summary:"Get task details"`
	Id     string `json:"id" v:"required"`
}

type GetTaskRes struct {
	g.Meta `mime:"application/json"`
	*Task
	*common.BaseResponse
}

type CancelTaskReq struct {
	g.Meta `path:"/v1/tasks/{id}/cancel" tags:"Tasks" method:"post" summary:"Cancel a pending or running task"`
	Id     string `json:"id" v:"required"`
}

type CancelTaskRes struct {
	g.Meta `mime:"application/json"`
	*common.BaseResponse
}

type RetryTaskReq struct {
	g.Meta `path:"/v1/tasks/{id}/retry" tags:"Tasks" method:"post" summary:"Retry a failed task"`
	Id     string `json:"id" v:"required"`
}

type RetryTaskRes struct {
	g.Meta `mime:"application/json"`
	*Task
	*common.BaseResponse
}
