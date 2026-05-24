// Package moderation реализует кэшированный гейт глобального флага модерации,
// хранящегося в platform_setting.
package moderation

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// SettingStorage описывает контракт к таблице platform_setting.
type SettingStorage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, updatedBy int64) error
}

const (
	keyModerationEnabled = "moderation_enabled"
	keySystemUserID      = "system_user_id"

	valTrue  = "true"
	valFalse = "false"
)

// ErrSystemUserNotConfigured возвращается, если в platform_setting нет system_user_id.
var ErrSystemUserNotConfigured = errors.New("system_user_id is not set in platform_setting")

// Gate — потокобезопасный кэш флага модерации с TTL.
type Gate struct {
	store SettingStorage
	log   *slog.Logger
	ttl   time.Duration

	mu        sync.RWMutex
	enabled   bool
	expiresAt time.Time
}

// NewGate создаёт гейт. ttl — частота обновления кэша из БД.
func NewGate(store SettingStorage, log *slog.Logger, ttl time.Duration) *Gate {
	return &Gate{store: store, log: log, ttl: ttl}
}

// IsEnabled возвращает кэшированное значение флага. При истечении TTL обновляет
// его из БД; ошибка чтения логируется и возвращается последнее известное значение
// (по умолчанию false).
func (g *Gate) IsEnabled(ctx context.Context) bool {
	g.mu.RLock()
	if time.Now().Before(g.expiresAt) {
		v := g.enabled
		g.mu.RUnlock()
		return v
	}
	g.mu.RUnlock()

	g.refresh(ctx)

	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enabled
}

// SetEnabled сохраняет новое значение в БД и инвалидирует кэш.
func (g *Gate) SetEnabled(ctx context.Context, enabled bool, adminID int64) error {
	val := valFalse
	if enabled {
		val = valTrue
	}

	if err := g.store.Set(ctx, keyModerationEnabled, val, adminID); err != nil {
		return err
	}

	g.mu.Lock()
	g.enabled = enabled
	g.expiresAt = time.Now().Add(g.ttl)
	g.mu.Unlock()
	return nil
}

func (g *Gate) refresh(ctx context.Context) {
	v, err := g.store.Get(ctx, keyModerationEnabled)
	if err != nil {
		g.log.WarnContext(ctx, "failed to read moderation flag",
			slog.String("error", err.Error()),
		)
		g.mu.Lock()
		g.expiresAt = time.Now().Add(g.ttl)
		g.mu.Unlock()
		return
	}

	g.mu.Lock()
	g.enabled = v == valTrue
	g.expiresAt = time.Now().Add(g.ttl)
	g.mu.Unlock()
}

// LoadSystemUserID читает идентификатор системного пользователя из platform_setting.
// Должна вызываться один раз при старте сервиса.
func LoadSystemUserID(ctx context.Context, store SettingStorage) (int64, error) {
	v, err := store.Get(ctx, keySystemUserID)
	if err != nil {
		return 0, ErrSystemUserNotConfigured
	}

	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, ErrSystemUserNotConfigured
	}
	return id, nil
}
