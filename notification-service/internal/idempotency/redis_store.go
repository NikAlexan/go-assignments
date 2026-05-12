package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func idempotencyKey(id string) string {
	return "idempotency:notification:" + id
}

func (s *RedisStore) Seen(ctx context.Context, id string) (bool, error) {
	exists, err := s.client.Exists(ctx, idempotencyKey(id)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisStore) Mark(ctx context.Context, id string) error {
	return s.client.Set(ctx, idempotencyKey(id), "processed", idempotencyTTL).Err()
}
