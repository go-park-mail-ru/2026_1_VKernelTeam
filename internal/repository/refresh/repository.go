package refresh

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/cache"
)

// Cache defines the subset of cache abstraction methods needed by the Refresh Token repository
type Cache interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type Repository struct {
	cache Cache
}

func New(c Cache) *Repository {
	return &Repository{cache: c}
}

// SaveRefresh saves the refresh token to the cache
func (r *Repository) SaveRefresh(ctx context.Context, token string, userID int64, ttl time.Duration) error {
	key := "refresh:" + token
	val := strconv.FormatInt(userID, 10)
	if err := r.cache.Set(ctx, key, val, ttl); err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	return nil
}

// GetRefresh retrieves the associated user ID from the refresh cache
func (r *Repository) GetRefresh(ctx context.Context, token string) (int64, error) {
	key := "refresh:" + token
	val, err := r.cache.Get(ctx, key)
	if err != nil {
		if err == cache.ErrNotFound {
			return 0, fmt.Errorf("refresh token not found") // match previous behavior required by auth tests
		}
		return 0, fmt.Errorf("failed to get refresh token: %w", err)
	}
	
	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user id format in cache: %w", err)
	}
	return userID, nil
}

// DeleteRefresh removes the target refresh token from cache
func (r *Repository) DeleteRefresh(ctx context.Context, token string) error {
	key := "refresh:" + token
	if err := r.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}
