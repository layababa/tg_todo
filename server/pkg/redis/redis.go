package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr      string
	Namespace string
}

func New(ctx context.Context, cfg Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
	})

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return rdb, nil
}
