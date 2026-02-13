package dataimport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gaap-api/internal/crypto"
	"gaap-api/internal/dao"
	"gaap-api/internal/export"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

const (
	ImportDir = "/app/imports"
)

// ImportResult contains import operation results
type ImportResult struct {
	AccountsImported     int
	TransactionsImported int
	AccountsSkipped      int
	TransactionsSkipped  int
}

// ImportData imports data from an encrypted export file.
// It validates user ownership and prevents cross-user imports.
func ImportData(ctx context.Context, userId, filePath string) (*ImportResult, error) {
	// Verify zip checksum and extract encrypted data
	encryptedData, err := export.VerifyZipChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("file verification failed: %w", err)
	}

	// Derive decryption key for this user
	key, err := crypto.DeriveKey(userId, crypto.GetServerSecret())
	if err != nil {
		return nil, fmt.Errorf("failed to derive decryption key: %w", err)
	}

	// Decrypt the data
	jsonData, err := crypto.Decrypt(encryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: file may belong to a different user or is corrupted")
	}

	// Parse the export data
	var exportData export.ExportData
	if err := json.Unmarshal(jsonData, &exportData); err != nil {
		return nil, fmt.Errorf("failed to parse export data: %w", err)
	}

	// Validate user ownership (double check after decryption)
	if exportData.UserId != userId {
		return nil, fmt.Errorf("import rejected: this export file belongs to a different user")
	}

	// Import data in a transaction
	result := &ImportResult{}
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		return executeImport(ctx, tx, userId, &exportData, result)
	})

	if err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	return result, nil
}

// executeImport performs the actual import within a database transaction
func executeImport(ctx context.Context, tx gdb.TX, userId string, data *export.ExportData, result *ImportResult) error {
	// Create a map of old account IDs to new account IDs
	accountIdMap := make(map[string]string)

	// Import accounts (parents first, then children)
	for _, acc := range data.Accounts {
		// Check if account already exists (by name, type, currency for this user)
		var existingId string
		val, err := tx.Model("accounts").
			Where("user_id", userId).
			Where("name", acc.Name).
			Where("type", acc.Type).
			Where("currency_code", acc.CurrencyCode).
			WhereNull("deleted_at").
			Fields("id").
			Value()
		if err != nil {
			return err
		}
		existingId = val.String()

		if existingId != "" {
			// Account exists, skip but map the ID
			accountIdMap[acc.Id] = existingId
			result.AccountsSkipped++
			continue
		}

		// Resolve parent ID if exists
		parentId := ""
		if acc.ParentId != "" {
			if newParentId, ok := accountIdMap[acc.ParentId]; ok {
				parentId = newParentId
			}
		}

		// Insert new account
		insertResult, err := tx.Model("accounts").Data(g.Map{
			"user_id":          userId,
			"parent_id":        parentId,
			"name":             acc.Name,
			"type":             acc.Type,
			"is_group":         boolToInt(acc.IsGroup),
			"balance":          0, // Start with zero balance, will be recalculated
			"currency_code":    acc.CurrencyCode,
			"default_child_id": "", // Will remap later if needed
			"date":             acc.Date,
			"number":           acc.Number,
			"remarks":          acc.Remarks,
			"created_at":       gtime.Now(),
			"updated_at":       gtime.Now(),
		}).Insert()
		if err != nil {
			return fmt.Errorf("failed to import account %s: %w", acc.Name, err)
		}

		newId, _ := insertResult.LastInsertId()
		accountIdMap[acc.Id] = fmt.Sprintf("%d", newId)
		result.AccountsImported++
	}

	// Import transactions
	for _, txn := range data.Transactions {
		// Map old account IDs to new ones
		fromAccountId := accountIdMap[txn.FromAccountId]
		toAccountId := accountIdMap[txn.ToAccountId]

		// Skip if required accounts don't exist
		if fromAccountId == "" && txn.FromAccountId != "" {
			result.TransactionsSkipped++
			continue
		}
		if toAccountId == "" && txn.ToAccountId != "" {
			result.TransactionsSkipped++
			continue
		}

		// Check for duplicate transaction
		var existingId string
		val, err := tx.Model("transactions").
			Where("user_id", userId).
			Where("date", txn.Date).
			Where("from_account_id", fromAccountId).
			Where("to_account_id", toAccountId).
			Where("balance_units", txn.BalanceUnits).
			Where("balance_nanos", txn.BalanceNanos).
			Where("currency_code", txn.CurrencyCode).
			WhereNull("deleted_at").
			Fields("id").
			Value()
		if err != nil {
			return err
		}
		existingId = val.String()

		if existingId != "" {
			result.TransactionsSkipped++
			continue
		}

		// Insert transaction
		_, err = tx.Model("transactions").Data(g.Map{
			"user_id":         userId,
			"date":            txn.Date,
			"from_account_id": fromAccountId,
			"to_account_id":   toAccountId,
			"balance_units":   txn.BalanceUnits,
			"balance_nanos":   txn.BalanceNanos,
			"currency_code":   txn.CurrencyCode,
			"note":            txn.Note,
			"type":            txn.Type,
			"created_at":      gtime.Now(),
			"updated_at":      gtime.Now(),
		}).Insert()
		if err != nil {
			return fmt.Errorf("failed to import transaction: %w", err)
		}

		result.TransactionsImported++
	}

	return nil
}

// boolToInt converts bool to int (for database storage)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// HasActiveImportTask checks if the user has a running import task.
// This is used to block account/transaction mutations during import.
func HasActiveImportTask(ctx context.Context, userId string) (bool, error) {
	count, err := dao.Tasks.Ctx(ctx).
		Where("user_id", userId).
		Where("type", model.TaskTypeDataImport).
		Where("status", model.TaskStatusRunning).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserIdFromContext extracts user ID from context
func GetUserIdFromContext(ctx context.Context) string {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	return userId
}

// SaveUploadedFile saves an uploaded import file to the import directory
func SaveUploadedFile(fileName string, data []byte) (string, error) {
	if err := os.MkdirAll(ImportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create import directory: %w", err)
	}

	filePath := filepath.Join(ImportDir, fileName)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save import file: %w", err)
	}

	return filePath, nil
}

// CleanupImport removes an import file
func CleanupImport(filePath string) error {
	return os.Remove(filePath)
}
