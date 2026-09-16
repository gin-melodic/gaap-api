package boot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gaap-api/internal/logic/reconciliation"
	"gaap-api/internal/observability"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// EnvStartupReconciliation selects the startup reconciliation mode: block, warn or off.
const EnvStartupReconciliation = "GAAP_STARTUP_RECONCILIATION"

// StartupReconcile runs the read-only ledger reconciler before the API serves
// traffic. In blocking mode a database error or any discrepancy fails boot so
// Docker crash-loops until the ledger is fixed; warn mode logs at Error level
// and continues, while off skips the check entirely without touching the DB.
func StartupReconcile(ctx context.Context) error {
	raw := os.Getenv(EnvStartupReconciliation)
	mode := strings.ToLower(strings.TrimSpace(raw))

	switch mode {
	case "":
		if IsProduction() {
			mode = "block"
		} else {
			mode = "warn"
		}
	case "off":
		g.Log().Info(ctx, "Startup reconciliation skipped: GAAP_STARTUP_RECONCILIATION=off")
		return nil
	case "block", "warn":
	default:
		if IsProduction() {
			return gerror.Newf("%s must be block, warn or off in production (got %q)", EnvStartupReconciliation, raw)
		}
		g.Log().Warningf(ctx, "%s has unknown value %q; falling back to warn", EnvStartupReconciliation, raw)
		mode = "warn"
	}

	report, err := reconciliation.Run(ctx)
	if err != nil {
		wrapped := gerror.Wrap(err, "startup reconciliation failed")
		observability.RecordEvent(observability.EventReconciliationStartupFailed, "error", wrapped.Error())
		if mode == "block" {
			return wrapped
		}
		g.Log().Errorf(ctx, "%v", wrapped.Error())
		return nil
	}

	if report.Passed {
		g.Log().Infof(ctx, "Startup reconciliation passed: %d account(s) and %d transaction(s) checked", report.AccountsChecked, report.TransactionsChecked)
		observability.SetLastReconciliation(report, "startup", time.Now())
		observability.RecordEvent(observability.EventReconciliationStartupPassed, "info", fmt.Sprintf("%d account(s), %d transaction(s) checked", report.AccountsChecked, report.TransactionsChecked))
		return nil
	}

	payload, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		payload = []byte(fmt.Sprintf("%+v", report))
	}
	g.Log().Errorf(ctx, "Startup ledger reconciliation failed: %s", payload)
	observability.SetLastReconciliation(report, "startup", time.Now())
	observability.RecordEvent(observability.EventReconciliationStartupFailed, "error", fmt.Sprintf("%d balance difference(s), %d issue(s)", len(report.Differences), len(report.Issues)))
	if mode == "warn" {
		return nil
	}
	return gerror.Newf("startup ledger reconciliation found %d balance difference(s) and %d issue(s); set %s=warn or off to override the gate", len(report.Differences), len(report.Issues), EnvStartupReconciliation)
}
