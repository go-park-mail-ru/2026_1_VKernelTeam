package blacklist

import (
	"context"
	"time"
)

// Cache defines the subset of cache abstraction methods needed by the Blacklist repository
type Cache interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Exists(ctx context.Context, key string) bool
}

type Repository struct {
	cache Cache
}

func New(c Cache) *Repository {
	return &Repository{cache: c}
}

// Add adds a JTI to the blacklist cache with the specified expiration
func (r *Repository) Add(jti string, exp time.Time) {
	ttl := time.Until(exp)
	if ttl <= 0 {
		return
	}
	key := "blacklist:" + jti
	_ = r.cache.Set(context.Background(), key, "1", ttl)
}

// Check checks if the JTI is blacklisted
func (r *Repository) Check(jti string) bool {
	key := "blacklist:" + jti
	return r.cache.Exists(context.Background(), key)
}
