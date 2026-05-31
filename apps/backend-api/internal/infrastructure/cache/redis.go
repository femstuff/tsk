package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedis(redisURL string) (*RedisCache, error) {
	redisURL = strings.TrimSpace(redisURL)
	if redisURL == "" {
		return nil, fmt.Errorf("redis url is empty")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Close()
}

func NewFromConfig(redisURL string) (Cache, func() error, error) {
	if strings.TrimSpace(redisURL) == "" {
		return NewNoop(), func() error { return nil }, nil
	}

	redisCache, err := NewRedis(redisURL)
	if err != nil {
		return nil, nil, err
	}

	return redisCache, redisCache.Close, nil
}
