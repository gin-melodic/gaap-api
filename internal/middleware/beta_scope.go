package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
)

func BetaScopeMiddleware(r *ghttp.Request) {
	if isProductionRuntime() && isDeferredBetaPath(r.URL.Path) {
		writeProtoError(r, http.StatusNotFound, "feature unavailable in beta", r.GetCtxVar("ale_key").String())
		return
	}
	r.Middleware.Next()
}

func isProductionRuntime() bool {
	environment := strings.ToLower(strings.TrimSpace(os.Getenv("GF_ENV")))
	if environment == "" {
		environment = strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	}
	return environment == "production" || environment == "prod"
}

func isDeferredBetaPath(path string) bool {
	if strings.HasPrefix(path, "/v1/task/") || strings.HasPrefix(path, "/v1/data/") {
		return true
	}
	if strings.HasPrefix(path, "/v1/user/update-") {
		return true
	}
	switch path {
	case "/v1/auth/generate2-f-a",
		"/v1/auth/enable2-f-a",
		"/v1/auth/disable2-f-a",
		"/v1/auth/update-password",
		"/v1/config/add-currency",
		"/v1/config/delete-currency":
		return true
	default:
		return false
	}
}
