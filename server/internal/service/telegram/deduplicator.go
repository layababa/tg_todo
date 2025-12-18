package telegram

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	pkgredis "github.com/layababa/tg_todo/server/pkg/redis"
)

const (
	// DefaultTTL is the expiration time for update keys (24 hours)
	DefaultTTL = 24 * time.Hour
)

// Deduplicator handles update deduplication
type Deduplicator interface {
	// IsDuplicate checks if the update ID has already been processed
	// If not duplicate, it marks it as processed (sets the key)
	IsDuplicate(ctx context.Context, updateID int64) (bool, error)
}

type redisDeduplicator struct {
	rdb *redis.Client
}

// NewDeduplicator creates a new Redis-based deduplicator
func NewDeduplicator(rdb *redis.Client) Deduplicator {
	return &redisDeduplicator{rdb: rdb}
}

// IsDuplicate checks if the update ID exists. If not, it sets it.
// Returns true if it was already present (duplicate).
func (d *redisDeduplicator) IsDuplicate(ctx context.Context, updateID int64) (bool, error) {
	key := pkgredis.GetTelegramUpdateKey(updateID)

	// SETNX key value
	// Returns true if key was set (not duplicate)
	// Returns false if key already exists (duplicate)
	set, err := d.rdb.SetNX(ctx, key, "1", DefaultTTL).Result()
	if err != nil {
		return false, err
	}

	return !set, nil
}
