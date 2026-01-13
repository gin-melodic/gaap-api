package utils

import (
	"context"
	"gaap-api/internal/middleware"

	"github.com/gogf/gf/v2/frame/g"
)

func RequireUserId(ctx context.Context) string {
	id, ok := ctx.Value(middleware.UserIdKey).(string)
	if !ok || id == "" {
		// In production, this error should be handled by Recovery middleware
		g.Log().Panicf(ctx, "user id not found in context")
	}
	return id
}
