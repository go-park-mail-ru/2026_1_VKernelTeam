package redis

import (
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
)

// ErrNotFound возвращается, когда запрашиваемый ключ не найден в кэше.
var ErrNotFound = errors.New("key not found in cache")

// RedisCache хранит пул соединений; его внутренние модули (view_cache, view_stream)
// работают напрямую с redis.Pool, поэтому только pool-обёртка тут нужна.
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

// Pool возвращает пул соединений Redis.
func (c *RedisCache) Pool() *redis.Pool {
	return c.pool
}

// Close закрывает пул соединений.
func (c *RedisCache) Close() error {
	return c.pool.Close()
}
