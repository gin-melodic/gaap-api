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
	"gaap-api/internal/service"
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
			if err := boot.ValidateProductionConfig(ctx); err != nil {
				return err
			}
			if err := boot.Migrate(ctx); err != nil {
				return err
			}
			if err := boot.InitRabbitMQ(ctx); err != nil {
				return err
			}
			// Initialize ALE (Application Layer Encryption)
			if err := boot.InitALE(ctx); err != nil {
				return err
			}

			// Account balances are committed atomically with transactions and must not
			// be silently rewritten during startup. Rebuild derived dashboard data
			// from the persisted source records instead.
			boot.WarmDashboardSnapshots(ctx)
			if err := service.DemoData().StartScheduler(ctx); err != nil {
				return err
			}

			s := g.Server()
			s.BindHandler("/v1/health/live", health.Live)
			s.BindHandler("/v1/health/ready", health.Ready)

			// Public routes (no authentication, no ALE - health checks, etc.)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					health.NewV1(),
				)
			})

			if !boot.IsProduction() {
				s.BindHandler("/v1/ws", ws.Handler)
			}

			// Protected routes (authentication required, with ALE using Session Key)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.ALEResponseMiddleware)
				group.Middleware(middleware.ALEMiddleware(middleware.ALEModeSession))
				group.Middleware(middleware.BetaScopeMiddleware)
				group.Middleware(middleware.AuthMiddleware)
				group.Bind(
					auth.NewV1(),
					config.NewV1(),
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
