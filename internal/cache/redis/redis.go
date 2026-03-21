package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/cache"
	"github.com/gomodule/redigo/redis"
)

type RedisCache struct {
	pool *redis.Pool
}

func New(addr string) *RedisCache {
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		},
	}
	return &RedisCache{pool: pool}
}

func (c *RedisCache) Close() error {
	return c.pool.Close()
}

func (c *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	conn := c.pool.Get()
	defer conn.Close()

	var err error
	if ttl > 0 {
		_, err = conn.Do("SET", key, value, "PX", int64(ttl/time.Millisecond))
	} else {
		_, err = conn.Do("SET", key, value)
	}
	
	if err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}
	return nil
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	conn := c.pool.Get()
	defer conn.Close()

	val, err := redis.String(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return "", cache.ErrNotFound
		}
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", key)
	if err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}
	return nil
}

func (c *RedisCache) Exists(ctx context.Context, key string) bool {
	conn := c.pool.Get()
	defer conn.Close()

	exists, _ := redis.Bool(conn.Do("EXISTS", key))
	return exists
}
