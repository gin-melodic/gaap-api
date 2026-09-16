package model

import (
	"time"

	"github.com/google/uuid"
)

// AccountDifference describes a persisted account balance that does not match
// the balance reconstructed from active transactions.
type AccountDifference struct {
	AccountID  uuid.UUID `json:"accountId"`
	UserID     uuid.UUID `json:"userId"`
	Name       string    `json:"name"`
	Type       int       `json:"type"`
	Currency   string    `json:"currency"`
	Actual     string    `json:"actual"`
	Expected   string    `json:"expected"`
	Difference string    `json:"difference"`
}

// Report is the result of a read-only reconciliation run.
type Report struct {
	Passed              bool                `json:"passed"`
	AccountsChecked     int                 `json:"accountsChecked"`
	TransactionsChecked int                 `json:"transactionsChecked"`
	Differences         []AccountDifference `json:"differences"`
	Issues              []string            `json:"issues"`
}

// OpsEvent is a single entry in the recent-events ring buffer.
type OpsEvent struct {
	Time   time.Time `json:"time"`
	Level  string    `json:"level"`
	Event  string    `json:"event"`
	Detail string    `json:"detail"`
}
