package export

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gaap-api/internal/crypto"
	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/os/gtime"
)

const (
	ExportVersion = "1.0"
	ExportDir     = "/app/exports"
)

// DateRange represents the export date range
type DateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// AccountExport represents an exported account
type AccountExport struct {
	Id             string  `json:"id"`
	ParentId       string  `json:"parentId,omitempty"`
	Name           string  `json:"name"`
	Type           int     `json:"type"`
	IsGroup        bool    `json:"isGroup"`
	CurrencyCode   string  `json:"currencyCode"`
	BalanceUnits   int64   `json:"balanceUnits"`
	BalanceNanos   int     `json:"balanceNanos"`
	BalanceDecimal float64 `json:"balanceDecimal"`
	DefaultChildId string  `json:"defaultChildId,omitempty"`
	Date           string  `json:"date,omitempty"`
	Number         string  `json:"number,omitempty"`
	Remarks        string  `json:"remarks,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// TransactionExport represents an exported transaction
type TransactionExport struct {
	Id             string  `json:"id"`
	Date           string  `json:"date"`
	FromAccountId  string  `json:"fromAccountId"`
	ToAccountId    string  `json:"toAccountId"`
	CurrencyCode   string  `json:"currencyCode"`
	BalanceUnits   int64   `json:"balanceUnits"`
	BalanceNanos   int     `json:"balanceNanos"`
	BalanceDecimal float64 `json:"balanceDecimal"`
	Note           string  `json:"note,omitempty"`
	Type           int     `json:"type"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// ExportData is the complete export structure
type ExportData struct {
	Version      string              `json:"version"`
	UserId       string              `json:"userId"`
	ExportedAt   string              `json:"exportedAt"`
	DateRange    DateRange           `json:"dateRange"`
	Accounts     []AccountExport     `json:"accounts"`
	Transactions []TransactionExport `json:"transactions"`
}

// ExportResult contains export operation results
type ExportResult struct {
	FilePath             string
	FileName             string
	FileSize             int64
	AccountsExported     int
	TransactionsExported int
}

// CreateExport creates an encrypted export file for the specified user and date range.
// Returns the file path, file name, and counts.
func CreateExport(ctx context.Context, userId, startDate, endDate string) (*ExportResult, error) {
	// Validate date range
	if err := validateDateRange(startDate, endDate); err != nil {
		return nil, err
	}

	// Fetch accounts for user
	var accounts []entity.Accounts
	err := dao.Accounts.Ctx(ctx).
		Where("user_id", userId).
		WhereNull("deleted_at").
		Scan(&accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	// Fetch transactions for user within date range
	var transactions []entity.Transactions
	err = dao.Transactions.Ctx(ctx).
		Where("user_id", userId).
		WhereBetween("date", startDate, endDate).
		WhereNull("deleted_at").
		Scan(&transactions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	// Build export data
	exportData := ExportData{
		Version:      ExportVersion,
		UserId:       userId,
		ExportedAt:   gtime.Now().Format("Y-m-d H:i:s"),
		DateRange:    DateRange{StartDate: startDate, EndDate: endDate},
		Accounts:     convertAccounts(accounts),
		Transactions: convertTransactions(transactions),
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Derive encryption key for this user
	key, err := crypto.DeriveKey(userId, crypto.GetServerSecret())
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Encrypt the JSON data
	encryptedData, err := crypto.Encrypt(jsonData, key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Calculate MD5 checksum of encrypted data
	md5Hash := md5.Sum(encryptedData)
	checksum := hex.EncodeToString(md5Hash[:])

	// Create export directory if not exists
	if err := os.MkdirAll(ExportDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate unique filename
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("gaap_export_%s_%s.zip", userId[:8], timestamp)
	filePath := filepath.Join(ExportDir, fileName)

	// Create ZIP file
	if err := createZipFile(filePath, encryptedData, checksum); err != nil {
		return nil, fmt.Errorf("failed to create zip file: %w", err)
	}

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat zip file: %w", err)
	}

	return &ExportResult{
		FilePath:             filePath,
		FileName:             fileName,
		FileSize:             fileInfo.Size(),
		AccountsExported:     len(accounts),
		TransactionsExported: len(transactions),
	}, nil
}

// validateDateRange validates the export date range
func validateDateRange(startDate, endDate string) error {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start date format: %w", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid end date format: %w", err)
	}

	if end.Before(start) {
		return fmt.Errorf("end date must be after start date")
	}

	// Check max 3 years
	maxEnd := start.AddDate(3, 0, 0)
	if end.After(maxEnd) {
		return fmt.Errorf("date range cannot exceed 3 years")
	}

	return nil
}

// convertAccounts converts entity accounts to export format
func convertAccounts(accounts []entity.Accounts) []AccountExport {
	result := make([]AccountExport, 0, len(accounts))
	for _, a := range accounts {
		exp := AccountExport{
			Id:             a.Id.String(),
			ParentId:       a.ParentId.String(),
			Name:           a.Name,
			Type:           a.Type,
			IsGroup:        a.IsGroup,
			CurrencyCode:   a.CurrencyCode,
			BalanceUnits:   a.BalanceUnits,
			BalanceNanos:   a.BalanceNanos,
			BalanceDecimal: a.BalanceDecimal,
			DefaultChildId: a.DefaultChildId.String(),
			Number:         a.Number,
			Remarks:        a.Remarks,
		}
		if a.Date != nil {
			exp.Date = a.Date.Format("Y-m-d")
		}
		if a.CreatedAt != nil {
			exp.CreatedAt = a.CreatedAt.Format("Y-m-d H:i:s")
		}
		if a.UpdatedAt != nil {
			exp.UpdatedAt = a.UpdatedAt.Format("Y-m-d H:i:s")
		}
		result = append(result, exp)
	}
	return result
}

// convertTransactions converts entity transactions to export format
func convertTransactions(transactions []entity.Transactions) []TransactionExport {
	result := make([]TransactionExport, 0, len(transactions))
	for _, t := range transactions {
		exp := TransactionExport{
			Id:             t.Id.String(),
			FromAccountId:  t.FromAccountId.String(),
			ToAccountId:    t.ToAccountId.String(),
			CurrencyCode:   t.CurrencyCode,
			BalanceUnits:   t.BalanceUnits,
			BalanceNanos:   t.BalanceNanos,
			BalanceDecimal: t.BalanceDecimal,
			Note:           t.Note,
			Type:           t.Type,
		}
		if t.Date != nil {
			exp.Date = t.Date.Format("Y-m-d")
		}
		if t.CreatedAt != nil {
			exp.CreatedAt = t.CreatedAt.Format("Y-m-d H:i:s")
		}
		if t.UpdatedAt != nil {
			exp.UpdatedAt = t.UpdatedAt.Format("Y-m-d H:i:s")
		}
		result = append(result, exp)
	}
	return result
}

// createZipFile creates a zip file containing encrypted data and checksum
func createZipFile(filePath string, encryptedData []byte, checksum string) error {
	zipFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add encrypted data file
	dataWriter, err := zipWriter.Create("data.enc")
	if err != nil {
		return err
	}
	if _, err := dataWriter.Write(encryptedData); err != nil {
		return err
	}

	// Add checksum file
	checksumWriter, err := zipWriter.Create("checksum.md5")
	if err != nil {
		return err
	}
	if _, err := checksumWriter.Write([]byte(checksum)); err != nil {
		return err
	}

	return nil
}

// GetUserIdFromContext extracts user ID from context
func GetUserIdFromContext(ctx context.Context) string {
	userId, _ := ctx.Value(middleware.UserIdKey).(string)
	return userId
}

// CleanupExport removes an export file
func CleanupExport(filePath string) error {
	return os.Remove(filePath)
}

// VerifyZipChecksum verifies the MD5 checksum of a zip file's encrypted data
func VerifyZipChecksum(zipPath string) ([]byte, error) {
	// Open zip file
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()

	var encryptedData []byte
	var storedChecksum string

	for _, file := range zipReader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file in zip: %w", err)
		}

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to read file in zip: %w", err)
		}
		rc.Close()

		switch file.Name {
		case "data.enc":
			encryptedData = buf.Bytes()
		case "checksum.md5":
			storedChecksum = buf.String()
		}
	}

	if encryptedData == nil {
		return nil, fmt.Errorf("data.enc not found in zip")
	}
	if storedChecksum == "" {
		return nil, fmt.Errorf("checksum.md5 not found in zip")
	}

	// Verify checksum
	md5Hash := md5.Sum(encryptedData)
	calculatedChecksum := hex.EncodeToString(md5Hash[:])

	if calculatedChecksum != storedChecksum {
		return nil, fmt.Errorf("checksum mismatch: file may be corrupted")
	}

	return encryptedData, nil
}
