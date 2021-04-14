package prioritystore

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/kcz17/profiler/priority"
	"time"
)

var DefaultExpiry = 1 * time.Hour

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string, password string, db int) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

var ctx = context.Background()

func (r *RedisStore) Set(sessionID string, priority priority.Priority) error {
	return r.client.Set(ctx, sessionID, priority.String(), DefaultExpiry).Err()
}
