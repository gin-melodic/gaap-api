package boot

import (
	"context"
	"gaap-api/internal/redis"

	_ "github.com/gogf/gf/contrib/nosql/redis/v2"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/genv"
)

func InitRedis(ctx context.Context) {
	// Check for existing Redis configuration and return early to avoid re-initializing it.
	_, ok := gredis.GetConfig()
	if ok {
		// Configuration already set; skip re-initialization to prevent overwriting existing Redis settings.
		return
	}

	host := genv.Get("REDIS_HOST", "127.0.0.1").String()
	port := genv.Get("REDIS_PORT", "6379").String()
	pass := genv.Get("REDIS_PASSWORD", "").String()
	address := host + ":" + port

	g.Log().Infof(ctx, "Initializing Redis with host: %s, port: %s", host, port)

	commonConfig := &gredis.Config{
		Address:     address,
		Pass:        pass,
		Db:          0,
		IdleTimeout: 600,
	}

	// Register sync lock redis
	gredis.SetConfig(commonConfig, redis.RedisTypeSyncLock)

	// Register ale redis
	gredis.SetConfig(commonConfig, redis.RedisTypeAle)

	// Register cache Redis (using DB1 for isolation)
	cacheConfig := &gredis.Config{
		Address:     address,
		Pass:        pass,
		Db:          1,
		IdleTimeout: 600,
	}
	gredis.SetConfig(cacheConfig, redis.RedisTypeCache)

	g.Log().Info(ctx, "Redis initialized successfully")
}
