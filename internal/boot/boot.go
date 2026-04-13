package boot

import (
	"context"
	"os"
	"strings"

	"gaap-api/internal/ale"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/mq"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
)

// InitRabbitMQ initializes RabbitMQ connection and starts task worker
func InitRabbitMQ(ctx context.Context) {
	if err := mq.GetRabbitMQ().Connect(ctx); err != nil {
		g.Log().Warningf(ctx, "RabbitMQ connection failed (tasks will be processed on retry): %v", err)
		return
	}

	// Start task worker to consume queue messages
	go func() {
		if err := service.Task().StartWorker(ctx); err != nil {
			g.Log().Errorf(ctx, "Task worker failed: %v", err)
		}
	}()

	// Start dashboard snapshot worker to consume dashboard refresh events
	go func() {
		if err := dashboard.StartDashboardWorker(ctx); err != nil {
			g.Log().Errorf(ctx, "Dashboard worker failed: %v", err)
		}
	}()

	// Start periodic snapshot flush ticker (Redis → DB persistence)
	go dashboard.StartSnapshotFlushTicker(ctx)
}

// Migrate executes database migration and seeding on startup.
func Migrate(ctx context.Context) {
	g.Log().Info(ctx, "Starting database migration...")

	// 1. Execute Schema Migration
	if err := executeSqlFile(ctx, "manifest/sql/0-schema.sql"); err != nil {
		g.Log().Fatalf(ctx, "Failed to execute schema migration: %v", err)
	}
	g.Log().Info(ctx, "Schema migration completed.")

	// 2. Execute dashboard snapshots table migration
	if err := executeSqlFile(ctx, "manifest/sql/2025020801_dashboard_snapshots.sql"); err != nil {
		g.Log().Warningf(ctx, "Failed to execute dashboard snapshots migration: %v", err)
	}

	// 3. Check if seeding is needed (check if users table is empty)
	count, err := g.DB().Model("account_types").Count()
	if err != nil {
		// If table doesn't exist, it should have been created by schema.sql.
		// If it still fails, it's a fatal error.
		g.Log().Fatalf(ctx, "Failed to check account_types table: %v", err)
	}

	if count == 0 {
		g.Log().Info(ctx, "Database appears empty. Seeding test data...")
		if err := executeSqlFile(ctx, "manifest/sql/2025011501_init.sql"); err != nil {
			g.Log().Fatalf(ctx, "Failed to seed Init data: %v", err)
		}
		g.Log().Info(ctx, "Init data seeding completed.")
	} else {
		g.Log().Info(ctx, "Database already contains data. Skipping seeding.")
	}
}

// executeSqlFile reads a SQL file and executes its content.
func executeSqlFile(ctx context.Context, filePath string) error {
	if !gfile.Exists(filePath) {
		g.Log().Warningf(ctx, "SQL file not found: %s", filePath)
		return nil
	}

	content := gfile.GetContents(filePath)
	if content == "" {
		return nil
	}

	// Split by semicolon to execute multiple statements if needed,
	// but gdb.Exec might handle it depending on the driver.
	// For simplicity and robustness with some drivers, we might want to split.
	// However, PostgreSQL driver usually handles multiple statements in one Exec if properly formatted,
	// OR we can rely on gdb to handle it.
	// Let's try executing the whole content first. If it fails, we might need to split.
	// Actually, standard sql package often supports one statement per Exec.
	// Let's split by ';' for safety, but be careful with triggers/functions.
	// Given our schema is simple CREATE TABLEs, splitting by ';' is safer.

	statements := splitSqlStatements(content)
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			// Filter out comment lines
			lines := strings.Split(stmt, "\n")
			var validLines []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "--") {
					validLines = append(validLines, line)
				}
			}
			stmt = strings.Join(validLines, "\n")
			stmt = strings.TrimSpace(stmt)

			if stmt == "" {
				continue
			}
			if _, err := tx.Exec(stmt); err != nil {
				g.Log().Errorf(ctx, "Failed to execute statement: %s\nError: %v", stmt, err)
				return err
			}
		}
		return nil
	})
}

// InitDatabaseConfig initializes database configuration from environment variables.
func InitDatabaseConfig(ctx context.Context) {
	link := os.Getenv("DATABASE_LINK")
	if link != "" {
		gdb.SetConfigGroup("default", gdb.ConfigGroup{
			gdb.ConfigNode{
				Link:  link,
				Debug: true,
			},
		})
		return
	}

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	name := os.Getenv("POSTGRES_DB")

	// If host is not set, we assume environment variables are not used for DB config
	// and fall back to config.yaml (which might fail if it has placeholders).
	if host == "" {
		g.Log().Warning(ctx, "Neither DATABASE_LINK nor POSTGRES_HOST set, skipping environment variable configuration for database.")
		return
	}

	gdb.SetConfigGroup("default", gdb.ConfigGroup{
		gdb.ConfigNode{
			Host:  host,
			Port:  port,
			User:  user,
			Pass:  pass,
			Name:  name,
			Type:  "pgsql",
			Role:  "master",
			Debug: true,
		},
	})
}

// InitConfig loads environment variables from .env file if it exists.
func InitConfig(ctx context.Context) {
	wd, _ := os.Getwd()
	g.Log().Infof(ctx, "Current working directory: %s", wd)

	envFile := ".env"
	if gfile.Exists(envFile) {
		g.Log().Infof(ctx, "Loading environment variables from %s...", gfile.Abs(envFile))
		content := gfile.GetContents(envFile)
		content = strings.ReplaceAll(content, "\r\n", "\n")
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Basic unquoting if present
				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					value = value[1 : len(value)-1]
				}

				// Local development override: if running natively and host is set to a docker service name,
				// redirect to 127.0.0.1 since middleware is in Docker and exposed to localhost.
				if value == "postgres" || value == "redis" || value == "rabbitmq" {
					value = "127.0.0.1"
				}

				if os.Getenv(key) == "" {
					os.Setenv(key, value)
				}
			}
		}
	} else {
		g.Log().Warningf(ctx, "Environment file not found: %s", gfile.Abs(envFile))
	}
}

// InitALE initializes the Application Layer Encryption (ALE) system
func InitALE(ctx context.Context) {
	g.Log().Info(ctx, "Initializing ALE (Application Layer Encryption)...")

	// Validate bootstrap key is configured
	if _, err := ale.GetBootstrapKey(); err != nil {
		g.Log().Warningf(ctx, "ALE bootstrap key not configured: %v", err)
		g.Log().Warning(ctx, "ALE will not be available for auth endpoints until ALE_BOOTSTRAP_KEY is set")
	} else {
		g.Log().Info(ctx, "ALE bootstrap key validated successfully")
	}
}
