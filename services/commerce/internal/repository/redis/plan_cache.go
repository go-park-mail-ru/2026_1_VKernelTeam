// Package redis — Redis-кэш для commerce-сервиса (тарифы продвижения).
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	planCacheKey = "commerce:promotion:plans:v1"
	planCacheTTL = 5 * time.Minute
)

// ErrCacheMiss — ключ отсутствует или TTL вышел.
var ErrCacheMiss = errors.New("plan cache miss")

// PlanCache хранит сериализованный список активных тарифов.
type PlanCache struct {
	pool *redis.Pool
	log  *slog.Logger
}

// NewPlanCache принимает существующий пул (например, из catalog) или создаёт собственный.
// Здесь используем собственный — каждый сервис владеет своим Redis-клиентом.
func NewPlanCache(addr string, log *slog.Logger) *PlanCache {
	pool := &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		},
	}
	return &PlanCache{pool: pool, log: log}
}

// NewPlanCacheWithPool позволяет переиспользовать готовый пул (используется в тестах).
func NewPlanCacheWithPool(pool *redis.Pool, log *slog.Logger) *PlanCache {
	return &PlanCache{pool: pool, log: log}
}

// Close закрывает пул.
func (c *PlanCache) Close() error {
	return c.pool.Close()
}

// Get возвращает закэшированные тарифы или ErrCacheMiss.
func (c *PlanCache) Get(ctx context.Context) ([]models.PromotionPlan, error) {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	val, err := redis.String(conn.Do("GET", planCacheKey))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("PlanCache.Get: %w", err)
	}

	var plans []models.PromotionPlan
	if err := json.Unmarshal([]byte(val), &plans); err != nil {
		c.log.WarnContext(ctx, "failed to unmarshal plan cache, treating as miss",
			slog.String("error", err.Error()),
		)
		return nil, ErrCacheMiss
	}
	return plans, nil
}

// Set кладёт тарифы в кэш на planCacheTTL.
func (c *PlanCache) Set(ctx context.Context, plans []models.PromotionPlan) error {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	payload, err := json.Marshal(plans)
	if err != nil {
		return fmt.Errorf("PlanCache.Set: marshal: %w", err)
	}

	_, err = conn.Do("SET", planCacheKey, payload, "PX", int64(planCacheTTL/time.Millisecond))
	if err != nil {
		return fmt.Errorf("PlanCache.Set: redis: %w", err)
	}
	return nil
}

// Invalidate удаляет ключ. Вызывается при изменении тарифов.
func (c *PlanCache) Invalidate(ctx context.Context) error {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	if _, err := conn.Do("DEL", planCacheKey); err != nil {
		return fmt.Errorf("PlanCache.Invalidate: %w", err)
	}
	return nil
}
