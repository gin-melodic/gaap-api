package model

import "github.com/gogf/gf/v2/os/gtime"

// Task status constants
const (
	TaskStatusPending   = "PENDING"
	TaskStatusRunning   = "RUNNING"
	TaskStatusCompleted = "COMPLETED"
	TaskStatusFailed    = "FAILED"
	TaskStatusCancelled = "CANCELLED"
)

// Task type constants
const (
	TaskTypeAccountMigration = "ACCOUNT_MIGRATION"
)

// Task represents a background task
type Task struct {
	Id             string      `json:"id"`
	UserId         string      `json:"userId"`
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

// TaskCreateInput for creating a new task
type TaskCreateInput struct {
	UserId  string      `orm:"user_id"`
	Type    string      `orm:"type"`
	Payload interface{} `orm:"payload"`
}

// TaskQueryInput for querying tasks
type TaskQueryInput struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// AccountMigrationPayload for account migration task
type AccountMigrationPayload struct {
	AccountId        string            `json:"accountId"`
	ChildAccountIds  []string          `json:"childAccountIds,omitempty"`
	MigrationTargets map[string]string `json:"migrationTargets"` // currency -> targetAccountId
}

// AccountMigrationResult for account migration task result
type AccountMigrationResult struct {
	TransactionsMigrated int    `json:"transactionsMigrated"`
	BalancesMerged       int    `json:"balancesMerged"`
	AccountsDeleted      int    `json:"accountsDeleted"`
	Error                string `json:"error,omitempty"`
}
