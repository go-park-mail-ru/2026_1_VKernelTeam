package refresh

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/redis"
)

// Cache определяет интерфейс для взаимодействия с кэшем, используемым для хранения refresh токенов.
type Cache interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

// Repository обеспечивает хранение refresh-токенов в кэше.
type Repository struct {
	cache Cache
}

// New создаёт Repository поверх переданного кэша.
func New(c Cache) *Repository {
	return &Repository{cache: c}
}

// SaveRefresh сохраняет refresh токен в кэше с привязкой к userID и временем жизни (TTL)
func (r *Repository) SaveRefresh(ctx context.Context, token string, userID int64, ttl time.Duration) error {
	key := "refresh:" + token
	val := strconv.FormatInt(userID, 10)
	if err := r.cache.Set(ctx, key, val, ttl); err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

// GetRefresh получает связанный с refresh токеном userID из кэша
func (r *Repository) GetRefresh(ctx context.Context, token string) (int64, error) {
	key := "refresh:" + token
	val, err := r.cache.Get(ctx, key)
	if err != nil {
		if err == redis.ErrNotFound {
			return 0, fmt.Errorf("refresh token not found")
		}
		return 0, fmt.Errorf("failed to get refresh token: %w", err)
	}

	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id format in cache: %w", err)
	}
	return userID, nil
}

// DeleteRefresh удаляет refresh токен из кэша.
func (r *Repository) DeleteRefresh(ctx context.Context, token string) error {
	key := "refresh:" + token
	if err := r.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}
