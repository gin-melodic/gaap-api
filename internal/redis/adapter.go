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
	RedisTypeCache    RedisGroupType = "cache" // Dedicated to data caching
)

// GetRedisClient returns a Redis client for the specified group type.
// Returns error if Redis is not configured or connection fails (graceful degradation).
func GetRedisClient(ctx context.Context, groupType RedisGroupType) (*gredis.Redis, error) {
	g.Log().Debugf(ctx, "Getting Redis client for group: %s", groupType)

	// Check if Redis is configured for this group
	config, ok := gredis.GetConfig(groupType)
	if !ok || config.Address == "" {
		g.Log().Debugf(ctx, "Redis not configured for group: %s", groupType)
		return nil, gerror.Newf("redis not configured for group: %s", groupType)
	}

	client := g.Redis(groupType)
	if _, err := client.Do(ctx, "PING"); err != nil {
		g.Log().Errorf(ctx, "Failed to connect to redis [%s]: %v", groupType, err)
		return nil, gerror.Wrap(err, "failed to connect to redis")
	}
	return client, nil
}

// GetCacheClient returns a Redis client dedicated to caching (convenience method).
// Returns error if Redis is not configured or unavailable (callers should handle graceful degradation).
func GetCacheClient(ctx context.Context) (*gredis.Redis, error) {
	return GetRedisClient(ctx, RedisTypeCache)
}
