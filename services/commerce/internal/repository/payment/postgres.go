// Package payment — лог пополнений кошелька через провайдера.
package payment

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
	opPaymentInsert  = "db.payment.Insert"
	opPaymentGetByID = "db.payment.GetByID"
)

var ErrPaymentNotFound = errors.New("payment not found")

// PgxPoolTx — пул с поддержкой транзакций.
type PgxPoolTx interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PaymentStorage отвечает за лог платежей.
type PaymentStorage struct {
	pool PgxPoolTx
	log  *slog.Logger
}

func NewPaymentStorage(pool PgxPoolTx, log *slog.Logger) *PaymentStorage {
	return &PaymentStorage{pool: pool, log: log}
}

// InsertTx сохраняет платёж внутри переданной транзакции.
func (s *PaymentStorage) InsertTx(
	ctx context.Context,
	tx pgx.Tx,
	userID, amount int64,
	status, provider string,
	providerRef *string,
) (models.Payment, error) {
	const query = `
		INSERT INTO payment (user_id, amount, status, provider, provider_ref)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, amount, status, provider, provider_ref, created_at, updated_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentInsert),
		slog.Int64("user_id", userID),
		slog.Int64("amount", amount),
		slog.String("status", status),
	)

	var p models.Payment
	err := tx.QueryRow(ctx, query, userID, amount, status, provider, providerRef).Scan(
		&p.ID, &p.UserID, &p.Amount, &p.Status, &p.Provider, &p.ProviderRef,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return models.Payment{}, fmt.Errorf("PaymentStorage.InsertTx: %w", err)
	}
	return p, nil
}

// GetByID возвращает платёж по идентификатору.
func (s *PaymentStorage) GetByID(ctx context.Context, id int64) (models.Payment, error) {
	const query = `
		SELECT id, user_id, amount, status, provider, provider_ref, created_at, updated_at
		FROM payment WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentGetByID),
		slog.Int64("payment_id", id),
	)

	var p models.Payment
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.Amount, &p.Status, &p.Provider, &p.ProviderRef,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Payment{}, ErrPaymentNotFound
		}
		return models.Payment{}, fmt.Errorf("PaymentStorage.GetByID: %w", err)
	}
	return p, nil
}
