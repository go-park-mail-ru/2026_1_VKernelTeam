package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

func newMockCache(conn *redigomock.Conn) *PlanCache {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) { return conn, nil },
	}
	return NewPlanCacheWithPool(pool, slog.Default())
}

func samplePlans() []models.PromotionPlan {
	now := time.Now().UTC().Truncate(time.Second)
	return []models.PromotionPlan{
		{ID: 1, Code: "boost_1d", Kind: "boost", DurationDays: 1, Price: 49, IsActive: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Code: "highlight_7d", Kind: "highlight", DurationDays: 7, Price: 99, IsActive: true, CreatedAt: now, UpdatedAt: now},
	}
}

func TestPlanCache_Get_Success(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	payload, err := json.Marshal(samplePlans())
	require.NoError(t, err)

	conn.Command("GET", planCacheKey).Expect(payload)

	got, err := cache.Get(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "boost_1d", got[0].Code)
}

func TestPlanCache_Get_Miss(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("GET", planCacheKey).Expect(nil)

	_, err := cache.Get(context.Background())
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestPlanCache_Get_BadJSON(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("GET", planCacheKey).Expect([]byte("not-a-json"))

	_, err := cache.Get(context.Background())
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestPlanCache_Get_RedisError(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("GET", planCacheKey).ExpectError(errors.New("connection refused"))

	_, err := cache.Get(context.Background())
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrCacheMiss)
}

func TestPlanCache_Set_Success(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	plans := samplePlans()
	payload, err := json.Marshal(plans)
	require.NoError(t, err)

	conn.Command("SET", planCacheKey, payload, "PX", int64(planCacheTTL/time.Millisecond)).Expect("OK")

	err = cache.Set(context.Background(), plans)
	assert.NoError(t, err)
}

func TestPlanCache_Set_Error(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.GenericCommand("SET").ExpectError(errors.New("redis down"))

	err := cache.Set(context.Background(), samplePlans())
	assert.Error(t, err)
}

func TestPlanCache_Invalidate(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("DEL", planCacheKey).Expect(int64(1))

	err := cache.Invalidate(context.Background())
	assert.NoError(t, err)
}

func TestPlanCache_Invalidate_Error(t *testing.T) {
	conn := redigomock.NewConn()
	cache := newMockCache(conn)

	conn.Command("DEL", planCacheKey).ExpectError(errors.New("boom"))

	err := cache.Invalidate(context.Background())
	assert.Error(t, err)
}

func TestNewPlanCache_ReturnsNonNil(t *testing.T) {
	c := NewPlanCache("127.0.0.1:6379", slog.Default())
	require.NotNil(t, c)
	require.NoError(t, c.Close())
}
