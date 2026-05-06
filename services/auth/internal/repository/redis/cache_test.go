package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockCache собирает RedisCache, чей пул отдаёт *redigomock.Conn вместо реального сокета.
func newMockCache(conn *redigomock.Conn) *RedisCache {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) { return conn, nil },
	}
	return &RedisCache{pool: pool}
}

func TestNew_ReturnsNonNilPool(t *testing.T) {
	c := New("127.0.0.1:6379")
	require.NotNil(t, c)
	assert.NotNil(t, c.Pool())
	_ = c.Close()
}

func TestSet_WithTTL(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("SET", "k", "v", "PX", int64(1000)).Expect("OK")

	err := cache.Set(context.Background(), "k", "v", time.Second)
	assert.NoError(t, err)
}

func TestSet_NoTTL(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("SET", "k", "v").Expect("OK")

	err := cache.Set(context.Background(), "k", "v", 0)
	assert.NoError(t, err)
}

func TestSet_Error(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("SET", "k", "v").ExpectError(errors.New("boom"))

	err := cache.Set(context.Background(), "k", "v", 0)
	assert.Error(t, err)
}

func TestGet_OK(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("GET", "k").Expect("v")

	val, err := cache.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", val)
}

func TestGet_NotFound(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("GET", "missing").Expect(nil)

	_, err := cache.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGet_OtherError(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("GET", "k").ExpectError(errors.New("boom"))

	_, err := cache.Get(context.Background(), "k")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNotFound)
}

func TestDelete_OK(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("DEL", "k").Expect(int64(1))

	err := cache.Delete(context.Background(), "k")
	assert.NoError(t, err)
}

func TestDelete_Error(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("DEL", "k").ExpectError(errors.New("boom"))

	err := cache.Delete(context.Background(), "k")
	assert.Error(t, err)
}

func TestExists_True(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("EXISTS", "k").Expect(int64(1))

	assert.True(t, cache.Exists(context.Background(), "k"))
}

func TestExists_False(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("EXISTS", "missing").Expect(int64(0))

	assert.False(t, cache.Exists(context.Background(), "missing"))
}

func TestClose(t *testing.T) {
	c := New("127.0.0.1:6379")
	assert.NoError(t, c.Close())
}
