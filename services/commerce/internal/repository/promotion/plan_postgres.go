// Package promotion — хранилище тарифов и купленных промо в PostgreSQL.
package promotion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	opPlanGetActive = "db.promotion.plan.GetActive"
	opPlanGetByCode = "db.promotion.plan.GetByCode"
	opPlanGetByID   = "db.promotion.plan.GetByID"
)

var (
	ErrPlanNotFound = errors.New("promotion plan not found")
	ErrPlanInactive = errors.New("promotion plan is inactive")
)

// PgxPoolTx — пул соединений с поддержкой транзакций (тот же контракт, что в cart-репо).
type PgxPoolTx interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PlanStorage отвечает за чтение каталога тарифов.
type PlanStorage struct {
	pool PgxPoolTx
	log  *slog.Logger
}

func NewPlanStorage(pool PgxPoolTx, log *slog.Logger) *PlanStorage {
	return &PlanStorage{pool: pool, log: log}
}

// GetActivePlans возвращает все активные тарифы.
func (s *PlanStorage) GetActivePlans(ctx context.Context) ([]models.PromotionPlan, error) {
	const query = `
		SELECT id, code, kind, duration_days, price, is_active, created_at, updated_at
		FROM promotion_plan
		WHERE is_active = true
		ORDER BY kind, duration_days
	`

	s.log.DebugContext(ctx, "executing query", slog.String("op", opPlanGetActive))

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to query active plans",
			slog.String("op", opPlanGetActive),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("PlanStorage.GetActivePlans: query: %w", err)
	}
	defer rows.Close()

	var plans []models.PromotionPlan
	for rows.Next() {
		var p models.PromotionPlan
		if err := rows.Scan(
			&p.ID, &p.Code, &p.Kind, &p.DurationDays, &p.Price,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("PlanStorage.GetActivePlans: scan: %w", err)
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PlanStorage.GetActivePlans: rows: %w", err)
	}
	if plans == nil {
		plans = []models.PromotionPlan{}
	}
	return plans, nil
}

// GetByCode возвращает тариф по уникальному коду.
// Возвращает ErrPlanNotFound, если тариф не найден, ErrPlanInactive — если деактивирован.
func (s *PlanStorage) GetByCode(ctx context.Context, code string) (models.PromotionPlan, error) {
	const query = `
		SELECT id, code, kind, duration_days, price, is_active, created_at, updated_at
		FROM promotion_plan
		WHERE code = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPlanGetByCode),
		slog.String("code", code),
	)

	var p models.PromotionPlan
	err := s.pool.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.Code, &p.Kind, &p.DurationDays, &p.Price,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PromotionPlan{}, ErrPlanNotFound
		}
		return models.PromotionPlan{}, fmt.Errorf("PlanStorage.GetByCode: %w", err)
	}
	if !p.IsActive {
		return p, ErrPlanInactive
	}
	return p, nil
}

// GetByID возвращает тариф по ID (используется для денормализации в ответах).
func (s *PlanStorage) GetByID(ctx context.Context, id int64) (models.PromotionPlan, error) {
	const query = `
		SELECT id, code, kind, duration_days, price, is_active, created_at, updated_at
		FROM promotion_plan
		WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPlanGetByID),
		slog.Int64("plan_id", id),
	)

	var p models.PromotionPlan
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Code, &p.Kind, &p.DurationDays, &p.Price,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PromotionPlan{}, ErrPlanNotFound
		}
		return models.PromotionPlan{}, fmt.Errorf("PlanStorage.GetByID: %w", err)
	}
	return p, nil
}
