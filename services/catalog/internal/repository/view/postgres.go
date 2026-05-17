package view

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opBatchInsertViews = "db.view.BatchInsertViews"
	opGetViewsCount    = "db.view.GetViewsCount"
)

// PgxPool интерфейс для пула соединений PostgreSQL.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// InsertEvent содержит данные для вставки просмотра.
type InsertEvent struct {
	ProductID int64
	UserID    *int64
	ViewedAt  time.Time
}

// ViewStorage отвечает за хранение просмотров в PostgreSQL.
type ViewStorage struct {
	pool PgxPool
	log  *slog.Logger
}

func NewViewStorage(pool PgxPool, log *slog.Logger) *ViewStorage {
	return &ViewStorage{pool: pool, log: log}
}

// BatchInsertViews вставляет пачку просмотров в product_view и обновляет views_count в product.
func (s *ViewStorage) BatchInsertViews(ctx context.Context, events []InsertEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", opBatchInsertViews, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Batch INSERT в product_view
	var sb strings.Builder
	sb.WriteString("INSERT INTO product_view (product_id, user_id, viewed_at) VALUES ")
	args := make([]any, 0, len(events)*3)
	for i, e := range events {
		if i > 0 {
			sb.WriteString(", ")
		}
		idx := i * 3
		fmt.Fprintf(&sb, "($%d, $%d, $%d)", idx+1, idx+2, idx+3)
		args = append(args, e.ProductID, e.UserID, e.ViewedAt)
	}

	if _, err := tx.Exec(ctx, sb.String(), args...); err != nil {
		s.log.ErrorContext(ctx, "failed to batch insert views",
			slog.String("op", opBatchInsertViews),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s: insert: %w", opBatchInsertViews, err)
	}

	// Группируем по product_id и обновляем views_count
	counts := make(map[int64]int64)
	for _, e := range events {
		counts[e.ProductID]++
	}

	for productID, count := range counts {
		const updateQuery = `UPDATE product SET views_count = views_count + $1 WHERE id = $2`
		if _, err := tx.Exec(ctx, updateQuery, count, productID); err != nil {
			s.log.ErrorContext(ctx, "failed to update views_count",
				slog.String("op", opBatchInsertViews),
				slog.Int64("product_id", productID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%s: update count: %w", opBatchInsertViews, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: commit: %w", opBatchInsertViews, err)
	}

	s.log.DebugContext(ctx, "batch insert views completed",
		slog.String("op", opBatchInsertViews),
		slog.Int("count", len(events)),
	)
	return nil
}

// GetViewsCount возвращает счётчик просмотров объявления из колонки views_count.
func (s *ViewStorage) GetViewsCount(ctx context.Context, productID int64) (int64, error) {
	const query = `SELECT views_count FROM product WHERE id = $1 AND deleted_at IS NULL`

	var count int64
	err := s.pool.QueryRow(ctx, query, productID).Scan(&count)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get views count",
			slog.String("op", opGetViewsCount),
			slog.Int64("product_id", productID),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("%s: %w", opGetViewsCount, err)
	}
	return count, nil
}
