// Package cache provides a thin, generic-friendly Redis wrapper used to
// cache-aside the segment summary/list reads — those run a handful of
// non-trivial queries against the remote AlefGym production database, so
// every admin dashboard refresh shouldn't re-hit it directly.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"centropy-affilate/internal/infrastructure/config"
)

var ErrMiss = errors.New("cache: miss")

type Cache struct {
	rdb *redis.Client
}

func New(cfg config.RedisConfig) *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
	})
	return &Cache{rdb: rdb}
}

func (c *Cache) Client() *redis.Client { return c.rdb }

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Close() error { return c.rdb.Close() }

// Get unmarshals a cached JSON value into T. Returns ErrMiss on cache miss
// so callers can distinguish "not cached" from a real Redis error.
func Get[T any](ctx context.Context, c *Cache, key string) (T, error) {
	var zero T
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return zero, ErrMiss
		}
		return zero, err
	}
	var val T
	if err := json.Unmarshal(raw, &val); err != nil {
		return zero, err
	}
	return val, nil
}

// Set marshals value as JSON and stores it with a TTL.
func Set[T any](ctx context.Context, c *Cache, key string, value T, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, raw, ttl).Err()
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}
