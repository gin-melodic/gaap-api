package model

import "github.com/google/uuid"

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

// TaskCreateInput for creating a new task
type TaskCreateInput struct {
	UserId  uuid.UUID   `orm:"user_id"`
	Type    TaskType    `orm:"type"`
	Payload interface{} `orm:"payload"`
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

// Task model for API responses
type Task struct {
	Id             uuid.UUID   `json:"id"`
	UserId         uuid.UUID   `json:"userId"`
	Type           TaskType    `json:"type"`
	Status         TaskStatus  `json:"status"`
	Payload        interface{} `json:"payload"`
	Result         interface{} `json:"result"`
	Progress       int         `json:"progress"`
	TotalItems     int         `json:"totalItems"`
	ProcessedItems int         `json:"processedItems"`
	StartedAt      interface{} `json:"startedAt"`
	CompletedAt    interface{} `json:"completedAt"`
	CreatedAt      interface{} `json:"createdAt"`
	UpdatedAt      interface{} `json:"updatedAt"`
}
