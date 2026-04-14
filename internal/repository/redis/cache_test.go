package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock/v3"
	"github.com/stretchr/testify/assert"

	myredis "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/redis"
)

type cacheWithMock struct {
	cache *myredis.RedisCache
	mock  *redigomock.Conn
}

func SetupMock(t *testing.T) *cacheWithMock {
	mockConn := redigomock.NewConn()

	// Создаем пул, который всегда возвращает наше моковое соединение
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return mockConn, nil
		},
	}

	cache := myredis.NewFromPool(pool)

	return &cacheWithMock{
		cache: cache,
		mock:  mockConn,
	}
}

func TestRedisCache_Set(t *testing.T) {
	c := SetupMock(t)
	ctx := context.Background()

	t.Run("set_with_ttl", func(t *testing.T) {
		key, val := "test_key", "test_val"
		ttl := time.Second * 10
		ms := int64(ttl / time.Millisecond)

		cmd := c.mock.Command("SET", key, val, "PX", ms).Expect("OK")

		err := c.cache.Set(ctx, key, val, ttl)

		assert.NoError(t, err)
		assert.True(t, cmd.Called())
	})

	t.Run("set_without_ttl", func(t *testing.T) {
		key, val := "no_ttl", "val"

		// Ожидаем обычный SET без дополнительных флагов
		c.mock.Command("SET", key, val).Expect("OK")

		err := c.cache.Set(ctx, key, "val", 0)
		assert.NoError(t, err)
	})
}

func TestRedisCache_Get(t *testing.T) {
	c := SetupMock(t)
	ctx := context.Background()

	t.Run("get_success", func(t *testing.T) {
		key, expectedVal := "user:1", "data"

		c.mock.Command("GET", key).Expect(expectedVal)

		val, err := c.cache.Get(ctx, key)

		assert.NoError(t, err)
		assert.Equal(t, expectedVal, val)
	})

	t.Run("get_not_found", func(t *testing.T) {
		key := "missing"

		c.mock.Command("GET", key).ExpectError(redis.ErrNil)

		val, err := c.cache.Get(ctx, key)

		assert.ErrorIs(t, err, myredis.ErrNotFound)
		assert.Empty(t, val)
	})
}

func TestRedisCache_Delete(t *testing.T) {
	c := SetupMock(t)
	ctx := context.Background()

	t.Run("delete_ok", func(t *testing.T) {
		key := "to_delete"
		c.mock.Command("DEL", key).Expect(int64(1))

		err := c.cache.Delete(ctx, key)
		assert.NoError(t, err)
	})
}

func TestRedisCache_Exists(t *testing.T) {
	c := SetupMock(t)
	ctx := context.Background()

	t.Run("key_exists", func(t *testing.T) {
		key := "existing_key"
		c.mock.Command("EXISTS", key).Expect(int64(1))

		exists := c.cache.Exists(ctx, key)
		assert.True(t, exists)
	})

	t.Run("key_not_exists", func(t *testing.T) {
		key := "ghost"
		c.mock.Command("EXISTS", key).Expect(int64(0))

		exists := c.cache.Exists(ctx, key)
		assert.False(t, exists)
	})
}
