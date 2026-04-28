package blacklist

import (
	"context"
	"time"
)

// Cache определяет интерфейс для взаимодействия с кэшем, используемым для хранения черного списка JWT.
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

// Add добавляет JTI в черный список с указанием времени истечения срока действия.
func (r *Repository) Add(jti string, exp time.Time) {
	ttl := time.Until(exp)
	if ttl <= 0 {
		return
	}
	key := "blacklist:" + jti
	_ = r.cache.Set(context.Background(), key, "1", ttl)
}

// Check проверяет, находится ли JTI в черном списке.
func (r *Repository) Check(jti string) bool {
	key := "blacklist:" + jti
	return r.cache.Exists(context.Background(), key)
}
