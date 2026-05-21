package promotion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	opPromotionInsert           = "db.promotion.Insert"
	opPromotionMaxActiveExpires = "db.promotion.MaxActiveExpires"
	opPromotionGetActiveByAd    = "db.promotion.GetActiveByAd"
	opPromotionListByUser       = "db.promotion.ListByUser"
	opPromotionGetByID          = "db.promotion.GetByID"
)

// ErrPromotionNotFound возвращается, когда купленное промо не найдено.
var ErrPromotionNotFound = errors.New("promotion not found")

// PromotionStorage отвечает за купленные промо.
type PromotionStorage struct {
	pool PgxPoolTx
	log  *slog.Logger
}

// NewPromotionStorage создаёт хранилище купленных промо.
func NewPromotionStorage(pool PgxPoolTx, log *slog.Logger) *PromotionStorage {
	return &PromotionStorage{pool: pool, log: log}
}

// Pool возвращает пул для transaction-методов usecase'а (purchase).
func (s *PromotionStorage) Pool() PgxPoolTx { return s.pool }

// InsertTx создаёт промо внутри переданной транзакции и возвращает заполненный объект.
func (s *PromotionStorage) InsertTx(
	ctx context.Context,
	tx pgx.Tx,
	productID, userID, planID int64,
	kind string,
	expiresAt time.Time,
	pricePaid int64,
) (models.Promotion, error) {
	const query = `
		INSERT INTO promotion (product_id, user_id, plan_id, kind, expires_at, price_paid)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, product_id, user_id, plan_id, kind, starts_at, expires_at, price_paid, created_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPromotionInsert),
		slog.Int64("product_id", productID),
		slog.Int64("user_id", userID),
		slog.String("kind", kind),
	)

	var p models.Promotion
	err := tx.QueryRow(ctx, query, productID, userID, planID, kind, expiresAt, pricePaid).Scan(
		&p.ID, &p.ProductID, &p.UserID, &p.PlanID, &p.Kind,
		&p.StartsAt, &p.ExpiresAt, &p.PricePaid, &p.CreatedAt,
	)
	if err != nil {
		return models.Promotion{}, fmt.Errorf("PromotionStorage.InsertTx: %w", err)
	}
	return p, nil
}

// MaxActiveExpiresTx возвращает максимальный expires_at среди активных промо
// заданного типа для объявления. Используется при продлении.
func (s *PromotionStorage) MaxActiveExpiresTx(
	ctx context.Context,
	tx pgx.Tx,
	productID int64,
	kind string,
) (time.Time, bool, error) {
	const query = `
		SELECT MAX(expires_at)
		FROM promotion
		WHERE product_id = $1 AND kind = $2 AND expires_at > CURRENT_TIMESTAMP
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPromotionMaxActiveExpires),
		slog.Int64("product_id", productID),
		slog.String("kind", kind),
	)

	var maxExpires *time.Time
	if err := tx.QueryRow(ctx, query, productID, kind).Scan(&maxExpires); err != nil {
		return time.Time{}, false, fmt.Errorf("PromotionStorage.MaxActiveExpiresTx: %w", err)
	}
	if maxExpires == nil {
		return time.Time{}, false, nil
	}
	return *maxExpires, true, nil
}

// GetActiveByAd возвращает все активные (expires_at > now) промо по объявлению.
func (s *PromotionStorage) GetActiveByAd(ctx context.Context, productID int64) ([]models.Promotion, error) {
	const query = `
		SELECT id, product_id, user_id, plan_id, kind, starts_at, expires_at, price_paid, created_at
		FROM promotion
		WHERE product_id = $1 AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC
	`
	return s.queryPromotions(ctx, opPromotionGetActiveByAd, query, productID)
}

// ListByUser возвращает историю промо пользователя с курсорной пагинацией.
// cursor — id последнего элемента предыдущей страницы (0 = первая страница).
func (s *PromotionStorage) ListByUser(
	ctx context.Context,
	userID int64,
	cursor int64,
	limit int,
) ([]models.Promotion, error) {
	const query = `
		SELECT id, product_id, user_id, plan_id, kind, starts_at, expires_at, price_paid, created_at
		FROM promotion
		WHERE user_id = $1 AND ($2 = 0 OR id < $2)
		ORDER BY id DESC
		LIMIT $3
	`
	return s.queryPromotions(ctx, opPromotionListByUser, query, userID, cursor, limit)
}

// GetByID возвращает промо по идентификатору. Используется для idempotency-возврата.
func (s *PromotionStorage) GetByID(ctx context.Context, id int64) (models.Promotion, error) {
	const query = `
		SELECT id, product_id, user_id, plan_id, kind, starts_at, expires_at, price_paid, created_at
		FROM promotion
		WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPromotionGetByID),
		slog.Int64("promotion_id", id),
	)

	var p models.Promotion
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.ProductID, &p.UserID, &p.PlanID, &p.Kind,
		&p.StartsAt, &p.ExpiresAt, &p.PricePaid, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Promotion{}, ErrPromotionNotFound
		}
		return models.Promotion{}, fmt.Errorf("PromotionStorage.GetByID: %w", err)
	}
	return p, nil
}

func (s *PromotionStorage) queryPromotions(
	ctx context.Context,
	op, query string,
	args ...any,
) ([]models.Promotion, error) {
	s.log.DebugContext(ctx, "executing query", slog.String("op", op))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("PromotionStorage.%s: query: %w", op, err)
	}
	defer rows.Close()

	var promos []models.Promotion
	for rows.Next() {
		var p models.Promotion
		if err := rows.Scan(
			&p.ID, &p.ProductID, &p.UserID, &p.PlanID, &p.Kind,
			&p.StartsAt, &p.ExpiresAt, &p.PricePaid, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("PromotionStorage.%s: scan: %w", op, err)
		}
		promos = append(promos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PromotionStorage.%s: rows: %w", op, err)
	}
	if promos == nil {
		promos = []models.Promotion{}
	}
	return promos, nil
}
