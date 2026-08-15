// Package cache provides the shared Redis client used for caching, sessions,
// and rate limiting across horizontally-scaled instances.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache: ping: %w", err)
	}
	return client, nil
}
