package store

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is a SessionStore backed by a Redis hash per session.
type Redis[T any] struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

// Compile-time check that *Redis[T] satisfies SessionStore[T].
var _ SessionStore[struct{}] = (*Redis[struct{}])(nil)

// NewRedis connects to Redis using the REDIS_URL connection string (e.g.
// "redis://:password@host:6379/0", or "rediss://" for TLS), verifies
// connectivity with a ping, and returns a store whose keys are namespaced by
// prefix and expire after ttl (ttl <= 0 disables expiry).
//
// REDIS_DB, when set, overrides the database number from the URL.
func NewRedis[T any](prefix string, ttl time.Duration) (*Redis[T], error) {
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	opt.Protocol = 3

	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		dbnum, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		opt.DB = dbnum
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Redis[T]{rdb: rdb, prefix: prefix, ttl: ttl}, nil
}

func (r *Redis[T]) key(cid string) string { return r.prefix + cid }

func (r *Redis[T]) Get(ctx context.Context, cid string) (T, bool, error) {
	var value T
	cmd := r.rdb.HGetAll(ctx, r.key(cid))
	m, err := cmd.Result()
	if err != nil {
		return value, false, err
	}
	if len(m) == 0 {
		return value, false, nil
	}
	if err := cmd.Scan(&value); err != nil {
		return value, false, err
	}
	return value, true, nil
}

func (r *Redis[T]) Set(ctx context.Context, cid string, value T) error {
	key := r.key(cid)
	pipe := r.rdb.TxPipeline()
	pipe.HSet(ctx, key, value)
	if r.ttl > 0 {
		pipe.Expire(ctx, key, r.ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Redis[T]) Delete(ctx context.Context, cid string) error {
	return r.rdb.Del(ctx, r.key(cid)).Err()
}

func (r *Redis[T]) Close() error {
	return r.rdb.Close()
}
