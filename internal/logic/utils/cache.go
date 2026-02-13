package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gaap-api/internal/redis"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
)

// GetOrLoad retrieves a single object from cache, or loads it via loader function if not cached.
// Automatically falls back to direct DB query if Redis is unavailable.
func GetOrLoad[T any](
	ctx context.Context,
	key string,
	ttl time.Duration,
	loader func(ctx context.Context) (*T, error),
) (*T, error) {
	// Get cache-dedicated Redis client
	client, err := redis.GetCacheClient(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "Redis cache unavailable, fallback to DB: %v", err)
		return loader(ctx) // Graceful degradation to direct query
	}

	// 1. Try to get from cache
	cached, err := client.Get(ctx, key)
	if err == nil && !cached.IsNil() {
		var result T
		if err := json.Unmarshal(cached.Bytes(), &result); err == nil {
			g.Log().Debugf(ctx, "Cache HIT: %s", key)
			return &result, nil
		}
		g.Log().Warningf(ctx, "Cache data unmarshal failed for key %s: %v", key, err)
	}

	g.Log().Debugf(ctx, "Cache MISS: %s", key)

	// 2. Execute loader
	result, err := loader(ctx)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	// 3. Async write to cache (non-blocking)
	go setCache(client, key, result, ttl)

	return result, nil
}

// BatchGetOrLoad retrieves multiple objects from cache in batch.
// Only loads missing items via loader function, minimizing DB queries.
func BatchGetOrLoad[T any](
	ctx context.Context,
	ids []interface{},
	keyPrefix string,
	ttl time.Duration,
	idExtractor func(*T) interface{},
	loader func(ctx context.Context, missedIDs []interface{}) ([]*T, error),
) ([]*T, error) {
	if len(ids) == 0 {
		return []*T{}, nil
	}

	client, err := redis.GetCacheClient(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "Redis cache unavailable, fallback to DB: %v", err)
		return loader(ctx, ids)
	}

	// Build cache keys
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("%s:%v", keyPrefix, id)
	}

	// Batch read from cache using individual Get calls
	// (MGet returns map which doesn't preserve order, so we iterate manually)
	cached := make(map[interface{}]*T)
	var missedIDs []interface{}

	for i, key := range keys {
		val, err := client.Get(ctx, key)
		if err == nil && val != nil && !val.IsNil() {
			var item T
			if err := json.Unmarshal(val.Bytes(), &item); err == nil {
				cached[ids[i]] = &item
				continue
			}
		}
		missedIDs = append(missedIDs, ids[i])
	}

	g.Log().Debugf(ctx, "Batch cache: %d hits, %d misses", len(cached), len(missedIDs))

	// Load missing items from DB
	if len(missedIDs) > 0 {
		items, err := loader(ctx, missedIDs)
		if err != nil {
			return nil, err
		}

		// Async batch write to cache
		go batchSetCache(client, keyPrefix, items, idExtractor, ttl)

		for _, item := range items {
			cached[idExtractor(item)] = item
		}
	}

	// Return in original order
	result := make([]*T, 0, len(ids))
	for _, id := range ids {
		if item, ok := cached[id]; ok {
			result = append(result, item)
		}
	}

	return result, nil
}

// InvalidateCache deletes specific cache keys.
// Returns nil if Redis is unavailable (cache invalidation failure should not block business).
func InvalidateCache(ctx context.Context, keys ...string) error {
	client, err := redis.GetCacheClient(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "Redis cache unavailable for invalidation: %v", err)
		return nil // Cache invalidation failure should not affect business
	}

	_, err = client.Del(ctx, keys...)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to invalidate cache: %v", err)
		return err
	}

	g.Log().Debugf(ctx, "Invalidated cache keys: %v", keys)
	return nil
}

// InvalidatePattern deletes cache keys matching a pattern (e.g., "user:*").
// Use with caution in production as KEYS command can be slow on large datasets.
func InvalidatePattern(ctx context.Context, pattern string) error {
	client, err := redis.GetCacheClient(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "Redis cache unavailable for pattern invalidation: %v", err)
		return nil
	}

	keys, err := client.Keys(ctx, pattern)
	if err != nil || len(keys) == 0 {
		return err
	}

	_, err = client.Del(ctx, keys...)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to invalidate cache pattern %s: %v", pattern, err)
		return err
	}

	g.Log().Infof(ctx, "Invalidated cache pattern %s: %d keys", pattern, len(keys))
	return nil
}

// RefreshCache forces a cache refresh by loading fresh data and updating the cache.
func RefreshCache[T any](
	ctx context.Context,
	key string,
	ttl time.Duration,
	loader func(ctx context.Context) (*T, error),
) error {
	result, err := loader(ctx)
	if err != nil {
		return err
	}

	client, err := redis.GetCacheClient(ctx)
	if err != nil {
		return err
	}

	return setCache(client, key, result, ttl)
}

// ========== Private helper functions ==========

func setCache[T any](client *gredis.Redis, key string, value *T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		g.Log().Errorf(context.Background(), "Failed to marshal cache data for key %s: %v", key, err)
		return err
	}

	err = client.SetEX(context.Background(), key, data, int64(ttl.Seconds()))
	if err != nil {
		g.Log().Errorf(context.Background(), "Failed to set cache for key %s: %v", key, err)
		return err
	}

	return nil
}

func batchSetCache[T any](
	client *gredis.Redis,
	keyPrefix string,
	items []*T,
	idExtractor func(*T) interface{},
	ttl time.Duration,
) {
	ctx := context.Background()

	for _, item := range items {
		id := idExtractor(item)
		key := fmt.Sprintf("%s:%v", keyPrefix, id)
		data, err := json.Marshal(item)
		if err != nil {
			g.Log().Errorf(ctx, "Failed to marshal item for key %s: %v", key, err)
			continue
		}

		err = client.SetEX(ctx, key, data, int64(ttl.Seconds()))
		if err != nil {
			g.Log().Errorf(ctx, "Failed to set cache for key %s: %v", key, err)
		}
	}
}
