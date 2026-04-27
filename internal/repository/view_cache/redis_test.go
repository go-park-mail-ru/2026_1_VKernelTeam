package viewcache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDedupTTL = 10 * time.Minute
	testCountTTL = 5 * time.Minute
)

func setup(t *testing.T) (*ViewCache, *redigomock.Conn) {
	t.Helper()
	mock := redigomock.NewConn()
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) { return mock, nil },
	}
	return New(pool, testDedupTTL, testCountTTL), mock
}

// ─── CheckAndSetDedup ────────────────────────────────────────────────────────

func TestCheckAndSetDedup_NewView(t *testing.T) {
	cache, mock := setup(t)
	ttlSec := int(testDedupTTL.Seconds())

	mock.Command("SET", "view:1:device-abc", "1", "EX", ttlSec, "NX").Expect("OK")

	isNew, err := cache.CheckAndSetDedup(context.Background(), 1, "device-abc")

	require.NoError(t, err)
	assert.True(t, isNew)
}

func TestCheckAndSetDedup_Duplicate(t *testing.T) {
	cache, mock := setup(t)
	ttlSec := int(testDedupTTL.Seconds())

	mock.Command("SET", "view:42:user-5", "1", "EX", ttlSec, "NX").ExpectError(redis.ErrNil)

	isNew, err := cache.CheckAndSetDedup(context.Background(), 42, "user-5")

	require.NoError(t, err)
	assert.False(t, isNew)
}

func TestCheckAndSetDedup_RedisError(t *testing.T) {
	cache, mock := setup(t)
	ttlSec := int(testDedupTTL.Seconds())

	mock.Command("SET", "view:1:x", "1", "EX", ttlSec, "NX").ExpectError(fmt.Errorf("READONLY"))

	isNew, err := cache.CheckAndSetDedup(context.Background(), 1, "x")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CheckAndSetDedup")
	assert.False(t, isNew)
}

// ─── IncrementCount ──────────────────────────────────────────────────────────

func TestIncrementCount_Success(t *testing.T) {
	cache, mock := setup(t)
	ttlSec := int(testCountTTL.Seconds())

	mock.Command("INCR", "views:count:7").Expect(int64(15))
	mock.Command("EXPIRE", "views:count:7", ttlSec).Expect(int64(1))

	count, err := cache.IncrementCount(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, int64(15), count)
}

func TestIncrementCount_RedisError(t *testing.T) {
	cache, mock := setup(t)

	mock.Command("INCR", "views:count:1").ExpectError(fmt.Errorf("OOM"))

	count, err := cache.IncrementCount(context.Background(), 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IncrementCount")
	assert.Equal(t, int64(0), count)
}

// ─── GetCount ────────────────────────────────────────────────────────────────

func TestGetCount_CacheHit(t *testing.T) {
	cache, mock := setup(t)

	mock.Command("GET", "views:count:3").Expect("250")

	count, found, err := cache.GetCount(context.Background(), 3)

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, int64(250), count)
}

func TestGetCount_CacheMiss(t *testing.T) {
	cache, mock := setup(t)

	mock.Command("GET", "views:count:99").ExpectError(redis.ErrNil)

	count, found, err := cache.GetCount(context.Background(), 99)

	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, int64(0), count)
}

func TestGetCount_ParseError(t *testing.T) {
	cache, mock := setup(t)

	mock.Command("GET", "views:count:5").Expect("not-a-number")

	count, found, err := cache.GetCount(context.Background(), 5)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
	assert.False(t, found)
	assert.Equal(t, int64(0), count)
}

func TestGetCount_RedisError(t *testing.T) {
	cache, mock := setup(t)

	mock.Command("GET", "views:count:1").ExpectError(fmt.Errorf("LOADING"))

	count, found, err := cache.GetCount(context.Background(), 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GetCount")
	assert.False(t, found)
	assert.Equal(t, int64(0), count)
}

// ─── SetCount ────────────────────────────────────────────────────────────────

func TestSetCount_Success(t *testing.T) {
	cache, mock := setup(t)
	ttlSec := int(testCountTTL.Seconds())

	mock.Command("SET", "views:count:10", "500", "EX", ttlSec).Expect("OK")

	err := cache.SetCount(context.Background(), 10, 500)

	assert.NoError(t, err)
}

func TestSetCount_RedisError(t *testing.T) {
	cache, mock := setup(t)
	ttlSec := int(testCountTTL.Seconds())

	mock.Command("SET", "views:count:1", "100", "EX", ttlSec).ExpectError(fmt.Errorf("READONLY"))

	err := cache.SetCount(context.Background(), 1, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SetCount")
}
