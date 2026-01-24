package model

import (
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/google/uuid"
)

// Task status constants
type TaskStatus = int

const (
	TaskStatusUnspecified TaskStatus = iota
	TaskStatusPending
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
	TaskStatusCancelled
)

// Task type constants
type TaskType = int

const (
	TaskTypeUnspecified TaskType = iota
	TaskTypeAccountMigration
	TaskTypeDataExport
	TaskTypeDataImport
)

// TaskPayload constraint for task payloads
type TaskPayload interface {
	AccountMigrationPayload | DataExportPayload | DataImportPayload | any
}

// TaskResult constraint for task results
type TaskResult interface {
	AccountMigrationResult | DataExportResult | DataImportResult | any
}

// TaskCreateInput for creating a new task
type TaskCreateInput[T TaskPayload] struct {
	UserId  uuid.UUID `orm:"user_id"`
	Type    TaskType  `orm:"type"`
	Payload T         `orm:"payload"`
}

// TaskQueryInput for querying tasks
type TaskQueryInput struct {
	Page   int        `json:"page"`
	Limit  int        `json:"limit"`
	Status TaskStatus `json:"status"`
	Type   TaskType   `json:"type"`
}

type Payload struct {
	UserId uuid.UUID `json:"userId"`
}

// AccountMigrationPayload for account migration task
type AccountMigrationPayload struct {
	*Payload
	AccountId        uuid.UUID            `json:"accountId"`
	ChildAccountIds  []uuid.UUID          `json:"childAccountIds,omitempty"`
	MigrationTargets map[string]uuid.UUID `json:"migrationTargets"` // currency -> targetAccountId
}

// AccountMigrationResult for account migration task result
type AccountMigrationResult struct {
	TransactionsMigrated int    `json:"transactionsMigrated"`
	BalancesMerged       int    `json:"balancesMerged"`
	AccountsDeleted      int    `json:"accountsDeleted"`
	Error                string `json:"error,omitempty"`
}

// DataExportPayload for data export task
type DataExportPayload struct {
	*Payload
	StartDate string `json:"startDate"` // YYYY-MM-DD
	EndDate   string `json:"endDate"`   // YYYY-MM-DD
}

// DataExportResult for data export task completion
type DataExportResult struct {
	FilePath             string `json:"filePath"`
	FileName             string `json:"fileName"`
	FileSize             int64  `json:"fileSize"`
	AccountsExported     int    `json:"accountsExported"`
	TransactionsExported int    `json:"transactionsExported"`
	Error                string `json:"error,omitempty"`
}

// DataImportPayload for data import task
type DataImportPayload struct {
	*Payload
	FileName string `json:"fileName"`
}

// DataImportResult for data import task completion
type DataImportResult struct {
	AccountsImported     int    `json:"accountsImported"`
	TransactionsImported int    `json:"transactionsImported"`
	AccountsSkipped      int    `json:"accountsSkipped"`
	TransactionsSkipped  int    `json:"transactionsSkipped"`
	Error                string `json:"error,omitempty"`
}

// TaskOutput model for API responses
type TaskOutput[P TaskPayload, R TaskResult] struct {
	Id             uuid.UUID   `json:"id"`
	UserId         uuid.UUID   `json:"userId"`
	Type           TaskType    `json:"type"`
	Status         TaskStatus  `json:"status"`
	Payload        P           `json:"payload"`
	Result         R           `json:"result"`
	Progress       int         `json:"progress"`
	TotalItems     int         `json:"totalItems"`
	ProcessedItems int         `json:"processedItems"`
	StartedAt      *gtime.Time `json:"startedAt"`
	CompletedAt    *gtime.Time `json:"completedAt"`
	CreatedAt      *gtime.Time `json:"createdAt"`
	UpdatedAt      *gtime.Time `json:"updatedAt"`
}
