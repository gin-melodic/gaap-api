package console

import (
	"fmt"
	"testing"

	"gaap-api/internal/mq"
	"gaap-api/internal/observability"
	"gaap-api/internal/service"
	"gaap-api/internal/testutil"

	"github.com/DATA-DOG/go-sqlmock"
)

var schemaColumns = map[string][]string{
	"users": {
		"id", "password", "email", "nickname", "avatar", "plan", "theme_id", "main_currency",
		"two_factor_secret", "two_factor_enabled", "created_at", "updated_at", "deleted_at",
	},
	"accounts": {
		"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos", "deleted_at",
	},
	"transactions": {
		"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type", "deleted_at",
	},
}

// initConsoleDB boots a fresh mock DB and registers schema expectations for the
// tables the test actually touches (the schema mock is once-per-process).
func initConsoleDB(t *testing.T, tables ...string) sqlmock.Sqlmock {
	t.Helper()
	mock, _ := testutil.InitMockDB(t)
	mock.MatchExpectationsInOrder(false)
	testutil.MockDBInit(mock)
	for _, table := range tables {
		testutil.MockMeta(mock, table, schemaColumns[table])
	}
	return mock
}

func TestStatusAssemblesObservationPayload(t *testing.T) {
	mock := initConsoleDB(t, "users")
	t.Cleanup(func() { mq.SetClient(nil) })

	mock.ExpectPing()
	mock.ExpectQuery(`SELECT COUNT\(.*\) FROM .users.`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))
	mock.ExpectQuery(`SELECT .* FROM .users. WHERE .*created_at.*`).
		WillReturnRows(sqlmock.NewRows([]string{"email", "created_at"}).
			AddRow("new@gaap.local", "2026-09-09T08:00:00Z"))

	mq.SetClient(&testutil.MockMQ{
		QueueInfos: map[string]testutil.MockQueueInfo{
			mq.QueueTasks:     {Depth: 7, Consumers: 2},
			mq.QueueDashboard: {Depth: 0, Consumers: 1},
		},
	})

	observability.ResetForTest()
	observability.RecordEvent(observability.EventALESignatureInvalid, "error", "/v1/account/create-account")
	observability.RecordHTTP("/v1/account/create-account", 500)

	status, err := service.Console().Status(t.Context())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status.Database.Status != "ok" {
		t.Fatalf("database = %+v, want ok", status.Database)
	}

	if status.RabbitMQ.Status != "ok" {
		t.Fatalf("rabbitmq = %+v, want ok", status.RabbitMQ)
	}
	if len(status.RabbitMQ.Queues) != 2 {
		t.Fatalf("rabbitmq.queues = %+v, want 2 entries", status.RabbitMQ.Queues)
	}
	foundTasks := false
	for _, q := range status.RabbitMQ.Queues {
		if q.Name == mq.QueueTasks && q.Depth == 7 && q.Consumers == 2 && q.Error == "" {
			foundTasks = true
		}
	}
	if !foundTasks {
		t.Fatalf("gaap.tasks queue not reported correctly: %+v", status.RabbitMQ.Queues)
	}

	if got := status.ALE.Total[observability.EventALESignatureInvalid]; got != 1 {
		t.Fatalf("ale total signature_invalid = %d, want 1", got)
	}
	if got := status.ALE.LastHour[observability.EventALESignatureInvalid]; got != 1 {
		t.Fatalf("ale lastHour signature_invalid = %d, want 1", got)
	}

	if got := status.HTTP.Total[observability.EventHTTP5xx]; got != 1 {
		t.Fatalf("http total 5xx = %d, want 1", got)
	}
	if got := status.HTTP.LastHour[observability.EventHTTP5xx]; got != 1 {
		t.Fatalf("http lastHour 5xx = %d, want 1", got)
	}

	if status.Users.Total != 12 {
		t.Fatalf("users.total = %d, want 12", status.Users.Total)
	}
	if len(status.Users.RecentSignups) != 1 || status.Users.RecentSignups[0].Email != "new@gaap.local" {
		t.Fatalf("users.recentSignups = %+v, want one signup for new@gaap.local", status.Users.RecentSignups)
	}

	if len(status.RecentEvents) < 2 {
		t.Fatalf("recentEvents = %+v, want at least the ALE and HTTP entries", status.RecentEvents)
	}

	if status.Reconciliation != nil {
		t.Fatalf("reconciliation = %+v, want nil (no startup run in tests)", status.Reconciliation)
	}

	if status.Server.GoVersion == "" || status.Server.StartedAt.IsZero() {
		t.Fatalf("server = %+v, want populated identity", status.Server)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestReconcileNowRecordsManualReport(t *testing.T) {
	mock := initConsoleDB(t, "accounts", "transactions")

	mock.ExpectBegin()
	mock.ExpectExec(`SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT .* FROM "?accounts"? WHERE .*deleted_at.*IS NULL.*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "name", "type", "currency_code", "balance_units", "balance_nanos",
		}))
	mock.ExpectQuery(`SELECT .* FROM "?transactions"? WHERE .*deleted_at.*IS NULL.*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "from_account_id", "to_account_id", "currency_code", "balance_units", "balance_nanos", "type",
		}))
	mock.ExpectCommit()

	observability.ResetForTest()
	report, err := service.Console().ReconcileNow(t.Context())
	if err != nil {
		t.Fatalf("ReconcileNow failed: %v", err)
	}
	if !report.Passed {
		t.Fatalf("empty ledger should reconcile: %+v", report)
	}

	snap := observability.LastReconciliation()
	if snap == nil || snap.Source != "manual" || snap.Report != report {
		t.Fatalf("last reconciliation snapshot = %+v, want manual run of this report", snap)
	}
	if got := observability.Counter(observability.EventReconciliationManualPassed); got != 1 {
		t.Fatalf("manual_passed counter = %d, want 1", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestStatusSurvivesDatabaseOutage(t *testing.T) {
	mock := initConsoleDB(t)
	mock.ExpectPing().WillReturnError(fmt.Errorf("db down"))

	status, err := service.Console().Status(t.Context())
	if err != nil {
		t.Fatalf("Status must not fail on probe errors: %v", err)
	}
	if status.Database.Status != "error" || status.Database.Error == "" {
		t.Fatalf("database = %+v, want error with message", status.Database)
	}
	if status.Users.Total != 0 {
		t.Fatalf("users.total = %d, want 0 when the count query fails", status.Users.Total)
	}
}
