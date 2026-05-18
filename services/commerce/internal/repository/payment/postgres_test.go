package payment

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/models"
)

func TestPaymentStorage_InsertTx_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewPaymentStorage(mock, slog.Default())
	ctx := context.Background()
	now := time.Now()
	ref := "mock-abc"

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO payment")).
		WithArgs(int64(1), int64(500), models.PaymentStatusSucceeded, models.PaymentProviderMock, &ref).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "user_id", "amount", "status", "provider", "provider_ref", "created_at", "updated_at",
			}).AddRow(int64(42), int64(1), int64(500), models.PaymentStatusSucceeded, models.PaymentProviderMock, &ref, now, now),
		)
	mock.ExpectCommit()

	tx, err := mock.Begin(ctx)
	require.NoError(t, err)

	got, err := storage.InsertTx(ctx, tx, 1, 500, models.PaymentStatusSucceeded, models.PaymentProviderMock, &ref)
	require.NoError(t, err)
	assert.Equal(t, int64(42), got.ID)
	assert.Equal(t, int64(500), got.Amount)
	assert.Equal(t, models.PaymentStatusSucceeded, got.Status)
	require.NotNil(t, got.ProviderRef)
	assert.Equal(t, ref, *got.ProviderRef)

	require.NoError(t, tx.Commit(ctx))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPaymentStorage_InsertTx_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewPaymentStorage(mock, slog.Default())
	ctx := context.Background()

	var nilRef *string
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO payment")).
		WithArgs(int64(1), int64(100), models.PaymentStatusSucceeded, models.PaymentProviderMock, nilRef).
		WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	tx, err := mock.Begin(ctx)
	require.NoError(t, err)

	_, err = storage.InsertTx(ctx, tx, 1, 100, models.PaymentStatusSucceeded, models.PaymentProviderMock, nil)
	assert.Error(t, err)

	require.NoError(t, tx.Rollback(ctx))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPaymentStorage_GetByID_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewPaymentStorage(mock, slog.Default())
	ctx := context.Background()
	now := time.Now()
	ref := "mock-xyz"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, amount, status")).
		WithArgs(int64(42)).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "user_id", "amount", "status", "provider", "provider_ref", "created_at", "updated_at",
			}).AddRow(int64(42), int64(1), int64(500), models.PaymentStatusSucceeded, models.PaymentProviderMock, &ref, now, now),
		)

	got, err := storage.GetByID(ctx, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), got.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPaymentStorage_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewPaymentStorage(mock, slog.Default())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = storage.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, ErrPaymentNotFound)
}
