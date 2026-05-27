// Package payment — лог пополнений кошелька через провайдера.
package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

const (
	opPaymentInsert            = "db.payment.Insert"
	opPaymentGetByID           = "db.payment.GetByID"
	opPaymentGetByProviderRef  = "db.payment.GetByProviderRef"
	opPaymentUpdateProviderRef = "db.payment.UpdateProviderRef"
	opPaymentUpdateStatus      = "db.payment.UpdateStatus"
	opPaymentListStale         = "db.payment.ListStalePending"
)

// ErrPaymentNotFound возвращается, когда платёж с указанным ID не найден.
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

// NewPaymentStorage создаёт PostgreSQL-репозиторий платежей.
func NewPaymentStorage(pool PgxPoolTx, log *slog.Logger) *PaymentStorage {
	return &PaymentStorage{pool: pool, log: log}
}

// InsertTx сохраняет платёж внутри переданной транзакции.
// Поля provider_ref и confirmation_url заполняются отдельно после ответа провайдера.
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
		RETURNING id, user_id, amount, status, provider, provider_ref,
		          confirmation_url, raw_provider_payload, created_at, updated_at
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
		&p.ConfirmationURL, &p.RawProviderPayload, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return models.Payment{}, fmt.Errorf("PaymentStorage.InsertTx: %w", err)
	}
	return p, nil
}

// GetByID возвращает платёж по идентификатору.
func (s *PaymentStorage) GetByID(ctx context.Context, id int64) (models.Payment, error) {
	const query = `
		SELECT id, user_id, amount, status, provider, provider_ref,
		       confirmation_url, raw_provider_payload, created_at, updated_at
		FROM payment WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentGetByID),
		slog.Int64("payment_id", id),
	)

	var p models.Payment
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.Amount, &p.Status, &p.Provider, &p.ProviderRef,
		&p.ConfirmationURL, &p.RawProviderPayload, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Payment{}, ErrPaymentNotFound
		}
		return models.Payment{}, fmt.Errorf("PaymentStorage.GetByID: %w", err)
	}
	return p, nil
}

// GetByProviderRefForUpdateTx внутри транзакции выбирает платёж по (provider, provider_ref) с FOR UPDATE.
// Используется webhook'ом и reconciler'ом для идемпотентного применения терминальных статусов.
func (s *PaymentStorage) GetByProviderRefForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	provider, providerRef string,
) (models.Payment, error) {
	const query = `
		SELECT id, user_id, amount, status, provider, provider_ref,
		       confirmation_url, raw_provider_payload, created_at, updated_at
		FROM payment
		WHERE provider = $1 AND provider_ref = $2
		FOR UPDATE
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentGetByProviderRef),
		slog.String("provider", provider),
	)

	var p models.Payment
	err := tx.QueryRow(ctx, query, provider, providerRef).Scan(
		&p.ID, &p.UserID, &p.Amount, &p.Status, &p.Provider, &p.ProviderRef,
		&p.ConfirmationURL, &p.RawProviderPayload, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Payment{}, ErrPaymentNotFound
		}
		return models.Payment{}, fmt.Errorf("PaymentStorage.GetByProviderRefForUpdateTx: %w", err)
	}
	return p, nil
}

// UpdateProviderRefTx проставляет provider_ref и confirmation_url по id (после ответа провайдера).
func (s *PaymentStorage) UpdateProviderRefTx(
	ctx context.Context,
	tx pgx.Tx,
	id int64,
	providerRef, confirmationURL string,
	rawPayload []byte,
) error {
	const query = `
		UPDATE payment
		SET provider_ref = $2, confirmation_url = $3, raw_provider_payload = $4,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentUpdateProviderRef),
		slog.Int64("payment_id", id),
	)

	tag, err := tx.Exec(ctx, query, id, providerRef, confirmationURL, rawPayload)
	if err != nil {
		return fmt.Errorf("PaymentStorage.UpdateProviderRefTx: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

// UpdateStatusTx меняет статус платежа и сохраняет последний payload провайдера.
func (s *PaymentStorage) UpdateStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	id int64,
	status string,
	rawPayload []byte,
) error {
	const query = `
		UPDATE payment
		SET status = $2, raw_provider_payload = COALESCE($3, raw_provider_payload),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentUpdateStatus),
		slog.Int64("payment_id", id),
		slog.String("status", status),
	)

	tag, err := tx.Exec(ctx, query, id, status, rawPayload)
	if err != nil {
		return fmt.Errorf("PaymentStorage.UpdateStatusTx: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

// ListStalePending возвращает pending-платежи указанного провайдера старше olderThan, для reconciler'а.
func (s *PaymentStorage) ListStalePending(
	ctx context.Context,
	provider string,
	olderThan time.Time,
	limit int,
) ([]models.Payment, error) {
	const query = `
		SELECT id, user_id, amount, status, provider, provider_ref,
		       confirmation_url, raw_provider_payload, created_at, updated_at
		FROM payment
		WHERE status = 'pending'
		  AND provider = $1
		  AND provider_ref IS NOT NULL
		  AND created_at < $2
		ORDER BY created_at
		LIMIT $3
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opPaymentListStale),
		slog.String("provider", provider),
	)

	rows, err := s.pool.Query(ctx, query, provider, olderThan, limit)
	if err != nil {
		return nil, fmt.Errorf("PaymentStorage.ListStalePending: %w", err)
	}
	defer rows.Close()

	var items []models.Payment
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Amount, &p.Status, &p.Provider, &p.ProviderRef,
			&p.ConfirmationURL, &p.RawProviderPayload, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("PaymentStorage.ListStalePending: scan: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PaymentStorage.ListStalePending: rows: %w", err)
	}
	return items, nil
}
