package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"gaap-api/internal/mq"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/database/gdb"
)

var (
	mockDB *sql.DB
)

type DriverMock struct {
	*pgsql.Driver
}

func (d *DriverMock) Open(config *gdb.ConfigNode) (db *sql.DB, err error) {
	fmt.Println("DriverMock.Open called")
	return mockDB, nil
}

func (d *DriverMock) New(core *gdb.Core, node *gdb.ConfigNode) (gdb.DB, error) {
	// Call the underlying driver's New method to initialize it
	db, err := d.Driver.New(core, node)
	if err != nil {
		return nil, err
	}
	// Wrap the returned DB (which is *pgsql.Driver) in our DriverMock
	if p, ok := db.(*pgsql.Driver); ok {
		return &DriverMock{Driver: p}, nil
	}
	return db, nil
}

func init() {
	// Register custom driver "mock"
	// We need to embed pgsql driver to inherit other methods
	var driver gdb.Driver = pgsql.New()
	if d, ok := driver.(*pgsql.Driver); ok {
		if err := gdb.Register("mock", &DriverMock{Driver: d}); err != nil {
			panic(err)
		}
	} else {
		panic("failed to cast pgsql driver")
	}
}

func InitMockDB(t *testing.T) (sqlmock.Sqlmock, gdb.DB) {
	var err error
	var mock sqlmock.Sqlmock
	mockDB, mock, err = sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	// Set mock configuration globally
	// We do this in InitMockDB to ensure it's set even if other things reset it
	// We use t.Name() in Extra to force gdb to create a new instance for each test
	// because gdb caches instances based on ConfigNode value.
	gdb.SetConfig(gdb.Config{
		"default": gdb.ConfigGroup{
			gdb.ConfigNode{
				Type:  "mock",
				Role:  "master",
				Debug: true,
				Extra: t.Name(),
			},
		},
	})

	// Debug
	cg, _ := gdb.GetConfigGroup("default")
	t.Logf("Config: %+v\n", cg)

	db, err := gdb.Instance("default")
	if err != nil {
		t.Fatalf("failed to get gdb instance: %v", err)
	}
	return mock, db
}

func MockDBInit(mock sqlmock.Sqlmock) {
	// Mock the PostgreSQL version query that GoFrame executes
	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("PostgreSQL mock"))

	// Mock the table names query that GoFrame executes
	mock.ExpectQuery("SELECT c.relname FROM pg_class c INNER JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = 'public' AND c.relkind IN \\('r', 'p'\\) ORDER BY c.relname").
		WillReturnRows(sqlmock.NewRows([]string{"relname"}))
}

func MockMeta(mock sqlmock.Sqlmock, tableName string, columns []string) {
	rows := sqlmock.NewRows([]string{"field", "type", "null", "key", "default_value", "comment", "length", "scale"})
	for _, col := range columns {
		key := ""
		if col == "id" || col == "code" {
			key = "pri"
		}
		rows.AddRow(col, "varchar", "NO", key, nil, "", nil, nil)
	}
	// Match the schema query. It's complex, so we use a loose regex.
	// It queries pg_attribute, pg_class, etc. and filters by relname.
	pattern := fmt.Sprintf("SELECT a.attname AS field(.*)WHERE c.relname = '%s'(.*)", tableName)
	mock.ExpectQuery(pattern).WillReturnRows(rows)
}

func MockVersion(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT version()").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("PostgreSQL mock"))
}

// MockMQ implements mq.Client for testing
type MockMQ struct {
	PublishedMessages []*mq.Message
}

func (m *MockMQ) IsConnected() bool                 { return true }
func (m *MockMQ) Connect(ctx context.Context) error { return nil }
func (m *MockMQ) Close() error                      { return nil }
func (m *MockMQ) Publish(ctx context.Context, queue string, msg *mq.Message) error {
	m.PublishedMessages = append(m.PublishedMessages, msg)
	return nil
}
func (m *MockMQ) Consume(ctx context.Context, queue string, handler func(ctx context.Context, msg *mq.Message) error) error {
	return nil
}
