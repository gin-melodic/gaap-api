package boot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gaap-api/internal/ale"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/mq"
	internalredis "gaap-api/internal/redis"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
)

// InitRabbitMQ initializes RabbitMQ and starts workers used by core runtime paths.
// Dashboard refresh depends on this connection, so startup fails closed when it
// cannot be established.
func InitRabbitMQ(ctx context.Context) error {
	client := mq.GetRabbitMQ()
	if err := client.Connect(ctx); err != nil {
		return gerror.Wrap(err, "RabbitMQ is unavailable")
	}
	startRabbitMQWorkers(ctx)
	go maintainRabbitMQConnection(ctx, client, time.Second, func() {
		startRabbitMQWorkers(ctx)
	})

	// Start periodic snapshot flush ticker (Redis → DB persistence) once. Worker
	// consumers are restarted separately whenever RabbitMQ reconnects.
	go dashboard.StartSnapshotFlushTicker(ctx)
	return nil
}

func startRabbitMQWorkers(ctx context.Context) {
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
}

func maintainRabbitMQConnection(
	ctx context.Context,
	client mq.Client,
	retryInterval time.Duration,
	onReconnect func(),
) {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if client.IsConnected() {
				continue
			}

			g.Log().Warning(ctx, "RabbitMQ connection lost; reconnecting")
			if err := client.Connect(ctx); err != nil {
				g.Log().Errorf(ctx, "RabbitMQ reconnect failed: %v", err)
				continue
			}

			g.Log().Info(ctx, "RabbitMQ reconnected; restarting consumers")
			onReconnect()
		}
	}
}

// Migrate executes database migration and seeding on startup.
func Migrate(ctx context.Context) error {
	g.Log().Info(ctx, "Starting database migration...")

	if _, err := g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return gerror.Wrap(err, "failed to initialize migration version table")
	}

	entries, err := os.ReadDir("manifest/sql")
	if err != nil {
		return gerror.Wrap(err, "failed to read migration directory")
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, file := range files {
		count, countErr := g.DB().Model("schema_migrations").Where("version", file).Count()
		if countErr != nil {
			return gerror.Wrapf(countErr, "failed to inspect migration %s", file)
		}
		if count > 0 {
			continue
		}
		if err := executeSqlFile(ctx, filepath.Join("manifest/sql", file), file); err != nil {
			return err
		}
		g.Log().Infof(ctx, "Applied migration %s", file)
	}
	return nil
}

// executeSqlFile reads a SQL file and executes its content.
func executeSqlFile(ctx context.Context, filePath, version string) error {
	if !gfile.Exists(filePath) {
		return gerror.Newf("SQL file not found: %s", filePath)
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
				return gerror.Wrapf(err, "migration %s failed", version)
			}
		}
		_, err := tx.Model("schema_migrations").Data(g.Map{"version": version}).Insert()
		return gerror.Wrap(err, fmt.Sprintf("failed to record migration %s", version))
	})
}

// InitDatabaseConfig initializes database configuration from environment variables.
func InitDatabaseConfig(ctx context.Context) {
	link := os.Getenv("DATABASE_LINK")
	if link != "" {
		gdb.SetConfigGroup("default", gdb.ConfigGroup{
			gdb.ConfigNode{
				Link:  link,
				Debug: !isProductionEnvironment(),
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
			Debug: !isProductionEnvironment(),
		},
	})
}

func isProductionEnvironment() bool {
	environment := strings.ToLower(strings.TrimSpace(os.Getenv("GF_ENV")))
	if environment == "" {
		environment = strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	}
	return environment == "production" || environment == "prod"
}

func IsProduction() bool {
	return isProductionEnvironment()
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
func InitALE(ctx context.Context) error {
	g.Log().Info(ctx, "Initializing ALE (Application Layer Encryption)...")

	if _, err := ale.GetBootstrapKey(); err != nil {
		if isProductionEnvironment() {
			return err
		}
		g.Log().Warningf(ctx, "ALE bootstrap key not configured: %v", err)
	}
	return nil
}

func ValidateProductionConfig(ctx context.Context) error {
	if !isProductionEnvironment() {
		return nil
	}
	if len(os.Getenv("JWT_SECRET")) < 32 {
		return gerror.New("JWT_SECRET must contain at least 32 characters in production")
	}
	if strings.TrimSpace(os.Getenv("TURNSTILE_SECRET")) == "" {
		return gerror.New("TURNSTILE_SECRET is required in production")
	}
	if strings.TrimSpace(os.Getenv("BETA_ALLOWED_EMAILS")) == "" {
		return gerror.New("BETA_ALLOWED_EMAILS is required in production")
	}
	for _, key := range []string{"RABBITMQ_HOST", "RABBITMQ_USER", "RABBITMQ_PASSWORD"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return gerror.Newf("%s is required in production", key)
		}
	}
	if _, err := internalredis.GetRedisClient(ctx, internalredis.RedisTypeAle); err != nil {
		return gerror.Wrap(err, "ALE Redis is unavailable")
	}
	return nil
}
