package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"gaap-api/internal/boot"
	"gaap-api/internal/controller/account"
	"gaap-api/internal/controller/auth"
	"gaap-api/internal/controller/config"
	"gaap-api/internal/controller/dashboard"
	"gaap-api/internal/controller/data"
	"gaap-api/internal/controller/health"
	"gaap-api/internal/controller/task"
	"gaap-api/internal/controller/transaction"
	"gaap-api/internal/controller/user"
	"gaap-api/internal/middleware"
	"gaap-api/internal/ws"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// Run database migration and seeding
			boot.InitConfig(ctx)
			boot.InitDatabaseConfig(ctx)
			boot.InitRedis(ctx)
			boot.Migrate(ctx)
			boot.InitRabbitMQ(ctx)

			// Initialize ALE (Application Layer Encryption)
			boot.InitALE(ctx)

			// Sync account balances (with Redis distributed lock)
			boot.SyncBalances(ctx)

			s := g.Server()

			// Public routes (no authentication, no ALE - health checks, etc.)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					health.NewV1(),
					config.NewV1(), // Config endpoints are public (currencies, etc.)
				)
			})

			// WebSocket route (special handling, no MiddlewareHandlerResponse)
			// Note: Route is /ws because Caddy's handle_path /api/* strips the /api prefix
			s.BindHandler("/v1/ws", ws.Handler)

			// Protected routes (authentication required, with ALE using Session Key)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.ALEResponseMiddleware)
				group.Middleware(middleware.ALEMiddleware(middleware.ALEModeSession))
				group.Middleware(middleware.AuthMiddleware)
				group.Bind(
					auth.NewV1(),
					user.NewV1(),
					account.NewV1(),
					transaction.NewV1(),
					dashboard.NewV1(),
					task.NewV1(),
					data.NewV1(),
				)
			})

			s.Run()
			return nil
		},
	}
)
