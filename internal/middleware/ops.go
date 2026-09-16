package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"gaap-api/internal/dao"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Admin-console (ops endpoint) configuration.
const (
	// EnvOpsPathCode overrides the random path segment used to mount the ops endpoints.
	EnvOpsPathCode = "CONSOLE_PATH_CODE"
	// EnvOpsAdminEmails is the ONLY source of admin identity: a comma-separated
	// allowlist of email addresses. Unset means nobody can use the console.
	EnvOpsAdminEmails = "CONSOLE_ADMIN_EMAILS"
	// EnvOpsAPIToken is an optional shared secret compared against the X-Ops-Token header.
	EnvOpsAPIToken = "CONSOLE_API_TOKEN"
	// HeaderOpsToken carries the optional shared secret.
	HeaderOpsToken = "X-Ops-Token"

	// DefaultOpsPathCode is the fallback random code (8 lowercase alphanumerics).
	DefaultOpsPathCode = "v7qk2xm9"

	opsRateLimitPerMinute = 60
	opsRateMapMaxEntries  = 10000
)

var opsPathCodePattern = regexp.MustCompile(`^[a-z0-9]{6,16}$`)

// ResolveOpsPathCode returns the ops path segment, failing boot when the
// override is set but invalid.
func ResolveOpsPathCode() (string, error) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(EnvOpsPathCode)))
	if raw == "" {
		return DefaultOpsPathCode, nil
	}
	if !opsPathCodePattern.MatchString(raw) {
		return raw, gerror.Newf("%s must match %s (got %q)", EnvOpsPathCode, opsPathCodePattern.String(), raw)
	}
	return raw, nil
}

// opsAdminEmails parses the allowlist into a normalized lowercase set.
// An unset or empty value yields an empty set (nobody is allowed).
func opsAdminEmails() map[string]bool {
	set := map[string]bool{}
	for _, part := range strings.Split(os.Getenv(EnvOpsAdminEmails), ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		if p != "" {
			set[p] = true
		}
	}
	return set
}

// OpsRateLimit enforces a fixed-window per-IP rate limit on the ops
// endpoints. The in-memory map is capped to bound memory under many clients.
func OpsRateLimit(r *ghttp.Request) {
	key := r.GetClientIp()
	if key == "" {
		key = "unknown"
	}

	now := time.Now()
	limited := false

	opsRateMu.Lock()
	if len(opsRateBuckets) > opsRateMapMaxEntries {
		for k, b := range opsRateBuckets {
			if now.Sub(b.windowStart) >= time.Minute {
				delete(opsRateBuckets, k)
			}
		}
	}
	b, ok := opsRateBuckets[key]
	if !ok || now.Sub(b.windowStart) >= time.Minute {
		opsRateBuckets[key] = &opsRateBucket{windowStart: now, count: 1}
	} else {
		b.count++
		limited = b.count > opsRateLimitPerMinute
	}
	opsRateMu.Unlock()

	if limited {
		writeOpsJSON(r, http.StatusTooManyRequests, "too many requests")
		return
	}
	r.Middleware.Next()
}

type opsRateBucket struct {
	windowStart time.Time
	count       int
}

var (
	opsRateMu      sync.Mutex
	opsRateBuckets = map[string]*opsRateBucket{}
)

// OpsAdmin authorizes the ops endpoints: the caller must hold a valid session
// token (set by AuthMiddleware) AND be listed in the admin allowlist. When
// CONSOLE_API_TOKEN is set it must additionally match X-Ops-Token exactly
// (constant-time compare). Errors are intentionally generic to avoid leaking
// which check failed.
func OpsAdmin(r *ghttp.Request) {
	ctx := r.Context()

	if expected := strings.TrimSpace(os.Getenv(EnvOpsAPIToken)); expected != "" {
		got := r.GetHeader(HeaderOpsToken)
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			writeOpsJSON(r, http.StatusForbidden, "forbidden")
			return
		}
	}

	uidVal := r.GetCtx().Value(UserIdKey)
	uid, _ := uidVal.(string)
	if uid == "" {
		writeOpsJSON(r, http.StatusForbidden, "forbidden")
		return
	}

	emailVal, err := dao.Users.Ctx(ctx).
		Where(dao.Users.Columns().Id, uid).
		Value(dao.Users.Columns().Email)
	if err != nil || emailVal.IsEmpty() {
		writeOpsJSON(r, http.StatusInternalServerError, "ops user lookup failed")
		return
	}
	email := strings.ToLower(strings.TrimSpace(emailVal.String()))

	allowed := opsAdminEmails()[email]
	if !allowed {
		// Admin email only in Debug logs; the response stays generic.
		g.Log().Debugf(ctx, "ops access denied for user %s (email %s)", uid, email)
		writeOpsJSON(r, http.StatusForbidden, "forbidden")
		return
	}

	r.Middleware.Next()
}

// writeOpsJSON emits a plain JSON ops response with no-store caching.
func writeOpsJSON(r *ghttp.Request, status int, message string) {
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJson(g.Map{"message": message})
}
