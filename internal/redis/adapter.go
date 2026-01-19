package redis

import (
	"context"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

type RedisGroupType = string

const (
	RedisTypeSyncLock RedisGroupType = "sync_lock"
	RedisTypeAle      RedisGroupType = "ale"
)

func GetRedisClient(ctx context.Context, groupType RedisGroupType) (*gredis.Redis, error) {
	g.Log().Infof(ctx, "Redis client initialized successfully")
	client := g.Redis(groupType)
	if _, err := client.Do(ctx, "PING"); err != nil {
		g.Log().Errorf(ctx, "Failed to connect to redis: %v", err)
		return nil, gerror.Wrap(err, "failed to connect to redis")
	}
	g.Log().Infof(ctx, "Redis client connected successfully")
	return client, nil
}
