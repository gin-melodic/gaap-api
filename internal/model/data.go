package model

import "github.com/google/uuid"

// DataExportInput for creating a data export task
type DataExportInput struct {
	StartDate string `json:"startDate"` // YYYY-MM-DD
	EndDate   string `json:"endDate"`   // YYYY-MM-DD
}

// DataExportOutput result of export task creation
type DataExportOutput struct {
	TaskId string `json:"taskId"`
}

// DataImportInput for creating a data import task
type DataImportInput struct {
	FileContent []byte `json:"fileContent"`
	FileName    string `json:"fileName"`
}

// DataImportOutput result of import task creation
type DataImportOutput struct {
	TaskId string `json:"taskId"`
}

// DataDownloadInput for downloading export file
type DataDownloadInput struct {
	TaskId uuid.UUID `json:"taskId"`
}
