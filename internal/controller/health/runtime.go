package health

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gaap-api/internal/mq"
	internalredis "gaap-api/internal/redis"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func Live(r *ghttp.Request) {
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteJson(g.Map{"status": "alive"})
}

func Ready(r *ghttp.Request) {
	ctx := r.Context()
	if _, err := g.DB().Exec(ctx, "SELECT 1"); err != nil {
		writeUnavailable(r)
		return
	}
	if _, err := internalredis.GetRedisClient(ctx, internalredis.RedisTypeAle); err != nil {
		writeUnavailable(r)
		return
	}
	if !mq.GetRabbitMQ().IsConnected() {
		writeUnavailable(r)
		return
	}

	entries, err := os.ReadDir("manifest/sql")
	if err != nil {
		writeUnavailable(r)
		return
	}
	expected := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".sql") {
			expected++
		}
	}
	applied, err := g.DB().Model("schema_migrations").Count()
	if err != nil || applied < expected {
		writeUnavailable(r)
		return
	}

	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteJson(g.Map{"status": "ready"})
}

func writeUnavailable(r *ghttp.Request) {
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteStatus(http.StatusServiceUnavailable)
	r.Response.WriteJson(g.Map{"status": "unavailable"})
}
