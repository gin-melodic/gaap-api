package console

import (
	"net/http"
	"time"

	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Status returns the admin-console observation payload as JSON.
// GET /v1/<code>/status
func Status(r *ghttp.Request) {
	status, err := service.Console().Status(r.Context())
	if err != nil {
		writeOpsJSON(r, http.StatusInternalServerError, "status unavailable")
		return
	}
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteJson(status)
}

// ReconcileNow triggers the read-only ledger reconciler on demand.
// POST /v1/<code>/reconcile
func ReconcileNow(r *ghttp.Request) {
	if r.Method != http.MethodPost {
		writeOpsJSON(r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	report, err := service.Console().ReconcileNow(r.Context())
	if err != nil {
		writeOpsJSON(r, http.StatusInternalServerError, "reconciliation unavailable")
		return
	}
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteJson(g.Map{
		"report":     report,
		"recordedAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func writeOpsJSON(r *ghttp.Request, status int, message string) {
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJson(g.Map{"message": message})
}
