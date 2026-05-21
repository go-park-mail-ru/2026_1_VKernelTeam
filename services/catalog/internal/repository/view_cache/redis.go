package viewcache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)

// ViewCache отвечает за дедупликацию просмотров и кэширование счётчика в Redis.
type ViewCache struct {
	pool          *redis.Pool
	dedupTTL      time.Duration
	countCacheTTL time.Duration
}

// New создаёт ViewCache с указанным пулом Redis и TTL дедупликации и счётчика.
func New(pool *redis.Pool, dedupTTL, countCacheTTL time.Duration) *ViewCache {
	return &ViewCache{
		pool:          pool,
		dedupTTL:      dedupTTL,
		countCacheTTL: countCacheTTL,
	}
}

// CheckAndSetDedup проверяет и устанавливает ключ дедупликации.
// Возвращает true, если просмотр новый (ключ создан).
func (c *ViewCache) CheckAndSetDedup(_ context.Context, productID int64, identifier string) (bool, error) {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	key := fmt.Sprintf("view:%d:%s", productID, identifier)
	ttlSeconds := int(c.dedupTTL.Seconds())

	// SET key 1 EX ttl NX — атомарно: ставит только если ключа нет
	reply, err := redis.String(conn.Do("SET", key, "1", "EX", ttlSeconds, "NX"))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil // дубликат
		}
		return false, fmt.Errorf("viewcache.CheckAndSetDedup: %w", err)
	}
	return reply == "OK", nil
}

// IncrementCount атомарно увеличивает кэшированный счётчик просмотров.
// Возвращает новое значение.
func (c *ViewCache) IncrementCount(_ context.Context, productID int64) (int64, error) {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	key := fmt.Sprintf("views:count:%d", productID)

	count, err := redis.Int64(conn.Do("INCR", key))
	if err != nil {
		return 0, fmt.Errorf("viewcache.IncrementCount: %w", err)
	}

	// Обновляем TTL при каждом инкременте
	ttlSeconds := int(c.countCacheTTL.Seconds())
	_, _ = conn.Do("EXPIRE", key, ttlSeconds)

	return count, nil
}

// GetCount возвращает кэшированный счётчик просмотров.
// Второе возвращаемое значение — true, если ключ найден.
func (c *ViewCache) GetCount(_ context.Context, productID int64) (int64, bool, error) {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	key := fmt.Sprintf("views:count:%d", productID)

	val, err := redis.String(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return 0, false, nil // cache miss
		}
		return 0, false, fmt.Errorf("viewcache.GetCount: %w", err)
	}

	count, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("viewcache.GetCount: parse: %w", err)
	}
	return count, true, nil
}

// SetCount устанавливает кэшированный счётчик (при cache miss после загрузки из БД).
func (c *ViewCache) SetCount(_ context.Context, productID int64, count int64) error {
	conn := c.pool.Get()
	defer func() { _ = conn.Close() }()

	key := fmt.Sprintf("views:count:%d", productID)
	ttlSeconds := int(c.countCacheTTL.Seconds())

	_, err := conn.Do("SET", key, strconv.FormatInt(count, 10), "EX", ttlSeconds)
	if err != nil {
		return fmt.Errorf("viewcache.SetCount: %w", err)
	}
	return nil
}
