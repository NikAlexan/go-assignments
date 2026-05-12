package repository

import (
	"context"
	"encoding/json"
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisOrderCache struct {
	client *redis.Client
}

func NewRedisOrderCache(client *redis.Client) *RedisOrderCache {
	return &RedisOrderCache{client: client}
}

func cacheKey(id string) string {
	return "order:" + id
}

func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	data, err := c.client.Get(ctx, cacheKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var order domain.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order, ttl time.Duration) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, cacheKey(order.ID), data, ttl).Err()
}

func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, cacheKey(id)).Err()
}
