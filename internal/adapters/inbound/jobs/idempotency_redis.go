package jobs

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	idempotencyKeyPrefix = "caliber:idem:"
	// defaultClaimTTL bounds how long an in-progress claim survives a crashed
	// worker before another replica may retry the task. It should exceed the
	// longest expected task runtime.
	defaultClaimTTL = 15 * time.Minute
	// defaultCompleteTTL is the duplicate-delivery suppression window: a completed
	// task's key keeps rejecting re-deliveries for this long.
	defaultCompleteTTL = 24 * time.Hour
)

// RedisIdempotencyStore is a shared, cross-process IdempotencyStore backed by
// Redis (CAL-157). Unlike MemoryIdempotencyStore — which only dedupes within one
// process — this gives exactly-once effects when the worker is scaled to several
// replicas: SET NX makes the claim atomic across the whole fleet, so a task
// re-delivered (or fanned to two workers) is processed once.
type RedisIdempotencyStore struct {
	client      redis.UniversalClient
	claimTTL    time.Duration
	completeTTL time.Duration
}

// NewRedisIdempotencyStore builds the store with sensible default TTLs.
func NewRedisIdempotencyStore(client redis.UniversalClient) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{
		client: client, claimTTL: defaultClaimTTL, completeTTL: defaultCompleteTTL,
	}
}

// Claim atomically reserves the key across the fleet. It returns true only for
// the first caller; a key already in-progress or completed returns false so the
// duplicate is skipped.
func (r *RedisIdempotencyStore) Claim(ctx context.Context, key string) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return false, errors.New("jobs: empty idempotency key")
	}
	ok, err := r.client.SetNX(ctx, redisIdemKey(key), "in-progress", r.claimTTL).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Complete marks a key done, refreshing it with the longer suppression TTL so
// later re-deliveries are rejected.
func (r *RedisIdempotencyStore) Complete(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("jobs: empty idempotency key")
	}
	return r.client.Set(ctx, redisIdemKey(key), "done", r.completeTTL).Err()
}

// Release drops an in-progress claim so a failed task can be retried, but keeps a
// completed key so it continues suppressing duplicates. The check-and-delete is
// a Lua script so it is atomic against a concurrent Complete on another worker.
func (r *RedisIdempotencyStore) Release(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("jobs: empty idempotency key")
	}
	const script = `if redis.call("GET", KEYS[1]) == "done" then return 0 else return redis.call("DEL", KEYS[1]) end`
	return r.client.Eval(ctx, script, []string{redisIdemKey(key)}).Err()
}

func redisIdemKey(key string) string { return idempotencyKeyPrefix + key }

var _ IdempotencyStore = (*RedisIdempotencyStore)(nil)
