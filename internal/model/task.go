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
	TaskTypeDataExport       = "DATA_EXPORT"
	TaskTypeDataImport       = "DATA_IMPORT"
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

// DataExportPayload for data export task
type DataExportPayload struct {
	UserId    string `json:"userId"`
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
	UserId   string `json:"userId"`
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
