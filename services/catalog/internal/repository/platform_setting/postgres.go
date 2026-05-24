// Package platform_setting предоставляет доступ к таблице platform_setting —
// глобальные настройки платформы вида ключ-значение.
package platform_setting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opGet = "db.platform_setting.Get"
	opSet = "db.platform_setting.Set"
)

// ErrSettingNotFound возвращается, когда ключа нет в таблице.
var ErrSettingNotFound = errors.New("platform setting not found")

// PgxPool — минимальный интерфейс пула pgx, используемый репозиторием.
type PgxPool interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

// Storage отвечает за чтение/запись настроек платформы.
type Storage struct {
	pool PgxPool
	log  *slog.Logger
}

// New создаёт хранилище.
func New(pool PgxPool, log *slog.Logger) *Storage {
	return &Storage{pool: pool, log: log}
}

// Get возвращает значение по ключу или ErrSettingNotFound.
func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	const query = `SELECT value FROM platform_setting WHERE key = $1`
	var value string
	err := s.pool.QueryRow(ctx, query, key).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSettingNotFound
		}
		s.log.ErrorContext(ctx, "failed to get setting",
			slog.String("op", opGet),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("platform_setting.Get: %w", err)
	}
	return value, nil
}

// Set перезаписывает значение по ключу.
func (s *Storage) Set(ctx context.Context, key, value string, updatedBy int64) error {
	const query = `
		INSERT INTO platform_setting (key, value, updated_at, updated_by)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (key) DO UPDATE
		   SET value      = EXCLUDED.value,
		       updated_at = EXCLUDED.updated_at,
		       updated_by = EXCLUDED.updated_by
	`
	if _, err := s.pool.Exec(ctx, query, key, value, updatedBy); err != nil {
		s.log.ErrorContext(ctx, "failed to set setting",
			slog.String("op", opSet),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("platform_setting.Set: %w", err)
	}
	return nil
}
