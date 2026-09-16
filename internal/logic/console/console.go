package console

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/reconciliation"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/mq"
	"gaap-api/internal/observability"
	redisAdapter "gaap-api/internal/redis"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
)

type sConsole struct{}

func init() {
	service.RegisterConsole(New())
}

func New() *sConsole {
	return &sConsole{}
}

// Status assembles the admin-console observation payload.
func (s *sConsole) Status(ctx context.Context) (out *model.OpsStatus, err error) {
	now := time.Now()
	startedAt := observability.ProcessStart()

	out = &model.OpsStatus{
		Server: model.OpsServerInfo{
			StartedAt: startedAt,
			UptimeSec: int64(now.Sub(startedAt).Seconds()),
			GoVersion: runtime.Version(),
			Env:       runtimeEnv(),
			Version:   apiVersion(),
		},
		Database: s.probeDatabase(ctx),
		Redis:    s.probeRedisGroups(ctx),
		RabbitMQ: s.probeRabbitMQ(ctx),
		HTTP:     eventStats(httpEventNames()),
		ALE:      eventStats(observability.ALEEvents),
		Auth:     eventStats(observability.AuthEvents),
	}
	out.Reconciliation = s.reconView()
	out.Users = s.usersView(ctx)
	out.RecentEvents = observability.RecentEvents(50)
	return out, nil
}

// ReconcileNow runs the read-only ledger reconciler on demand and records the
// result for the console.
func (s *sConsole) ReconcileNow(ctx context.Context) (out *model.Report, err error) {
	report, err := reconciliation.Run(ctx)
	if err != nil {
		observability.RecordEvent(observability.EventReconciliationManualFailed, "error", err.Error())
		return nil, err
	}
	observability.SetLastReconciliation(report, "manual", time.Now())
	if report.Passed {
		observability.RecordEvent(observability.EventReconciliationManualPassed, "info",
			fmt.Sprintf("manual run: %d account(s), %d transaction(s)", report.AccountsChecked, report.TransactionsChecked))
	} else {
		observability.RecordEvent(observability.EventReconciliationManualFailed, "error",
			fmt.Sprintf("manual run: %d balance difference(s), %d issue(s)", len(report.Differences), len(report.Issues)))
	}
	return report, nil
}

func httpEventNames() []string {
	return []string{
		observability.EventHTTPTotal,
		observability.EventHTTP2xx,
		observability.EventHTTP3xx,
		observability.EventHTTP4xx,
		observability.EventHTTP5xx,
	}
}

func eventStats(events []string) model.OpsEventStats {
	total := map[string]int64{}
	lastHour := map[string]int64{}
	for _, name := range events {
		total[name] = observability.Counter(name)
		lastHour[name] = observability.Since(name, 60)
	}
	return model.OpsEventStats{Total: total, LastHour: lastHour}
}

func (s *sConsole) probeDatabase(ctx context.Context) model.OpsDependencyStatus {
	start := time.Now()
	if err := g.DB().PingMaster(); err != nil {
		return model.OpsDependencyStatus{Status: "error", Error: err.Error()}
	}
	return model.OpsDependencyStatus{Status: "ok", LatencyMs: time.Since(start).Milliseconds()}
}

func (s *sConsole) probeRedisGroups(ctx context.Context) map[string]model.OpsDependencyStatus {
	groups := []string{
		redisAdapter.RedisTypeSyncLock,
		redisAdapter.RedisTypeAle,
		redisAdapter.RedisTypeCache,
	}
	out := make(map[string]model.OpsDependencyStatus, len(groups))
	for _, group := range groups {
		config, ok := gredis.GetConfig(group)
		if !ok || config.Address == "" {
			out[group] = model.OpsDependencyStatus{Status: "not_configured"}
			continue
		}
		start := time.Now()
		if _, err := redisAdapter.GetRedisClient(ctx, group); err != nil {
			out[group] = model.OpsDependencyStatus{Status: "error", Error: err.Error()}
			continue
		}
		out[group] = model.OpsDependencyStatus{Status: "ok", LatencyMs: time.Since(start).Milliseconds()}
	}
	return out
}

func (s *sConsole) probeRabbitMQ(ctx context.Context) model.OpsRabbitMQStatus {
	client := mq.GetRabbitMQ()
	if client == nil {
		return model.OpsRabbitMQStatus{Status: "not_configured", Queues: []model.OpsQueueStatus{}}
	}

	status := "ok"
	queues := make([]model.OpsQueueStatus, 0, 2)
	for _, name := range []string{mq.QueueTasks, mq.QueueDashboard} {
		q := model.OpsQueueStatus{Name: name}
		depth, consumers, err := client.QueueInfo(ctx, name)
		if err != nil {
			if status == "ok" {
				status = "error"
			}
			q.Error = err.Error()
		} else {
			q.Depth = depth
			q.Consumers = consumers
		}
		queues = append(queues, q)
	}
	if !client.IsConnected() && status == "ok" {
		status = "error"
	}
	return model.OpsRabbitMQStatus{Status: status, Queues: queues}
}

func (s *sConsole) reconView() *model.OpsReconView {
	snap := observability.LastReconciliation()
	if snap == nil {
		return nil
	}
	return &model.OpsReconView{At: snap.At, Source: snap.Source, Report: snap.Report}
}

func (s *sConsole) usersView(ctx context.Context) model.OpsUsersView {
	view := model.OpsUsersView{RecentSignups: []model.OpsSignupView{}}

	total, err := dao.Users.Ctx(ctx).
		WhereNull(dao.Users.Columns().DeletedAt).
		Count()
	if err == nil {
		view.Total = total
	}

	since := time.Now().AddDate(0, -7, 0)
	var rows []entity.Users
	err = dao.Users.Ctx(ctx).
		Where(dao.Users.Columns().CreatedAt+" > ?", since).
		WhereNull(dao.Users.Columns().DeletedAt).
		OrderDesc(dao.Users.Columns().CreatedAt).
		Limit(10).
		Scan(&rows)
	if err != nil {
		return view
	}
	for _, u := range rows {
		signup := model.OpsSignupView{Email: u.Email}
		if u.CreatedAt != nil {
			signup.CreatedAt = u.CreatedAt.Time
		}
		view.RecentSignups = append(view.RecentSignups, signup)
	}
	return view
}

func runtimeEnv() string {
	for _, key := range []string{"GF_ENV", "ENV"} {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv(key))); v != "" {
			return v
		}
	}
	return "development"
}

func apiVersion() string {
	if v := strings.TrimSpace(os.Getenv("GAAP_VERSION")); v != "" {
		return v
	}
	return "dev"
}
