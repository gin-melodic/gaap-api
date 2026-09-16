package cmd

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"gaap-api/internal/boot"
	"gaap-api/internal/controller/account"
	"gaap-api/internal/controller/auth"
	"gaap-api/internal/controller/config"
	"gaap-api/internal/controller/console"
	"gaap-api/internal/controller/dashboard"
	"gaap-api/internal/controller/data"
	"gaap-api/internal/controller/health"
	"gaap-api/internal/controller/task"
	"gaap-api/internal/controller/transaction"
	"gaap-api/internal/controller/user"
	"gaap-api/internal/middleware"
	"gaap-api/internal/observability"
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
			if err := boot.StartupReconcile(ctx); err != nil {
				return err
			}

			// Account balances are committed atomically with transactions and must not
			// be silently rewritten during startup. Rebuild derived dashboard data
			// from the persisted source records instead.
			boot.WarmDashboardSnapshots(ctx)
			// The online demo scheduler must never block boot on a bad baseline or a
			// transient infrastructure hiccup; log and keep serving so operators can
			// fix the environment without restarting the whole API.
			if err := service.DemoData().StartScheduler(ctx); err != nil {
				g.Log().Errorf(ctx, "Online demo scheduler failed to start: %v", err)
			}
			if err := service.ExchangeRate().StartScheduler(ctx); err != nil {
				return err
			}

			// The ops console mounts under a random path segment; an invalid
			// override must fail boot instead of serving a guessable URL.
			opsPathCode, err := middleware.ResolveOpsPathCode()
			if err != nil {
				return err
			}

			s := g.Server()
			s.BindHandler("/v1/health/live", health.Live)
			s.BindHandler("/v1/health/ready", health.Ready)

			// Count every HTTP response for the ops console observation window.
			s.BindHookHandler("/*", ghttp.HookAfterServe, func(r *ghttp.Request) {
				status := r.Response.Status
				if status == 0 {
					status = http.StatusOK
				}
				observability.RecordHTTP(r.URL.Path, status)
			})

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

			// Ops console: random path code, plain JSON, JWT + email allowlist.
			// AuthMiddleware runs first so unauthenticated traffic gets a generic
			// 401 (proto error, no ALE key in play); OpsAdmin adds the allowlist.
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.OpsRateLimit)
				group.Middleware(middleware.AuthMiddleware)
				group.Middleware(middleware.OpsAdmin)
				group.GET("/v1/"+opsPathCode+"/status", console.Status)
				// GET on the reconcile path answers 405 in the handler; the
				// handler stays the single place that enforces POST-only.
				group.GET("/v1/"+opsPathCode+"/reconcile", console.ReconcileNow)
				group.POST("/v1/"+opsPathCode+"/reconcile", console.ReconcileNow)
			})

			s.Run()
			return nil
		},
	}
)
