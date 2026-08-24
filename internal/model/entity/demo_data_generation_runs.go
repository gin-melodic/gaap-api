package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

type DemoDataGenerationRuns struct {
	UserId         uuid.UUID   `json:"userId" orm:"user_id"`
	BusinessDate   *gtime.Time `json:"businessDate" orm:"business_date"`
	GeneratedCount int         `json:"generatedCount" orm:"generated_count"`
	CreatedAt      *gtime.Time `json:"createdAt" orm:"created_at"`
}
