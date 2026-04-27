package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

// ErrNotFound возвращается, когда запрашиваемый ключ не найден в кэше.
var ErrNotFound = errors.New("key not found in cache")

type RedisCache struct {
	pool *redis.Pool
}

// NewFromPool используется для внедрения зависимостей (например, в тестах).
func NewFromPool(pool *redis.Pool) *RedisCache {
	return &RedisCache{pool: pool}
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

// Pool возвращает пул соединений Redis для прямого использования.
func (c *RedisCache) Pool() *redis.Pool {
	return c.pool
}

// Close закрывает пул соединений Redis, освобождая все ресурсы.
func (c *RedisCache) Close() error {
	return c.pool.Close()
}

// Set сохраняет значение в Redis с указанным ключом и временем жизни (TTL).
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
		if errors.Is(err, redis.ErrNil) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

// Delete удаляет значение из Redis по указанному ключу.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	conn := c.pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", key)
	if err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}
	return nil
}

// Exists проверяет, существует ли ключ в Redis, возвращая true, если ключ найден.
func (c *RedisCache) Exists(ctx context.Context, key string) bool {
	conn := c.pool.Get()
	defer conn.Close()

	exists, _ := redis.Bool(conn.Do("EXISTS", key))
	return exists
}
