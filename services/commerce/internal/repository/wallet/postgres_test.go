package wallet

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

const (
	colBalance   = "balance"
	colAmount    = "amount"
	colCreatedAt = "created_at"
)

func newStorage(t *testing.T) (*WalletStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	return NewWalletStorage(mock, slog.Default()), mock
}

func TestWalletStorage_Get_Success(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id, balance, updated_at FROM wallet")).
		WithArgs(int64(1)).
		WillReturnRows(
			pgxmock.NewRows([]string{"user_id", colBalance, "updated_at"}).
				AddRow(int64(1), int64(500), now),
		)

	got, err := storage.Get(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(500), got.Balance)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletStorage_Get_NotFound(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
		WithArgs(int64(2)).
		WillReturnError(pgx.ErrNoRows)

	_, err := storage.Get(context.Background(), 2)
	assert.ErrorIs(t, err, ErrWalletNotFound)
}

func TestWalletStorage_GetOrCreate(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO wallet")).
		WithArgs(int64(3)).
		WillReturnRows(
			pgxmock.NewRows([]string{"user_id", colBalance, "updated_at"}).
				AddRow(int64(3), int64(0), now),
		)

	got, err := storage.GetOrCreate(context.Background(), 3)
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.Balance)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletStorage_GetForUpdateTx_Success(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO wallet (user_id, balance) VALUES ($1, 0)")).
		WithArgs(int64(4)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(regexp.QuoteMeta("FOR UPDATE")).
		WithArgs(int64(4)).
		WillReturnRows(
			pgxmock.NewRows([]string{"user_id", colBalance, "updated_at"}).
				AddRow(int64(4), int64(700), now),
		)
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	got, err := storage.GetForUpdateTx(context.Background(), tx, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(700), got.Balance)

	require.NoError(t, tx.Commit(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletStorage_IncrementBalanceTx(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE wallet")).
		WithArgs(int64(1), int64(500)).
		WillReturnRows(pgxmock.NewRows([]string{colBalance}).AddRow(int64(500)))
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	newBalance, err := storage.IncrementBalanceTx(context.Background(), tx, 1, 500)
	require.NoError(t, err)
	assert.Equal(t, int64(500), newBalance)

	require.NoError(t, tx.Commit(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletStorage_DecrementBalanceTx(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE wallet")).
		WithArgs(int64(1), int64(199)).
		WillReturnRows(pgxmock.NewRows([]string{colBalance}).AddRow(int64(301)))
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	newBalance, err := storage.DecrementBalanceTx(context.Background(), tx, 1, 199)
	require.NoError(t, err)
	assert.Equal(t, int64(301), newBalance)

	require.NoError(t, tx.Commit(context.Background()))
}

func TestWalletStorage_InsertTransactionTx_Success(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	now := time.Now()
	refID := int64(42)
	key := "idem-1"

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO wallet_transaction")).
		WithArgs(int64(1), int64(500), models.WalletTxTypeTopup, &refID, &key).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "user_id", colAmount, "type", "reference_id", "idempotency_key", colCreatedAt,
			}).AddRow(int64(10), int64(1), int64(500), models.WalletTxTypeTopup, &refID, &key, now),
		)
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	got, err := storage.InsertTransactionTx(context.Background(), tx, 1, 500, models.WalletTxTypeTopup, &refID, &key)
	require.NoError(t, err)
	assert.Equal(t, int64(10), got.ID)
	require.NotNil(t, got.ReferenceID)
	assert.Equal(t, refID, *got.ReferenceID)
	require.NotNil(t, got.IdempotencyKey)
	assert.Equal(t, key, *got.IdempotencyKey)

	require.NoError(t, tx.Commit(context.Background()))
}

func TestWalletStorage_GetTransactionByIdempotencyKey(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		storage, mock := newStorage(t)
		defer mock.Close()

		now := time.Now()
		key := "idem-2"
		refID := int64(7)
		mock.ExpectQuery(regexp.QuoteMeta("FROM wallet_transaction")).
			WithArgs("idem-2").
			WillReturnRows(
				pgxmock.NewRows([]string{
					"id", "user_id", colAmount, "type", "reference_id", "idempotency_key", colCreatedAt,
				}).AddRow(int64(10), int64(1), int64(500), models.WalletTxTypeTopup, &refID, &key, now),
			)

		got, err := storage.GetTransactionByIdempotencyKey(context.Background(), "idem-2")
		require.NoError(t, err)
		assert.Equal(t, int64(10), got.ID)
	})

	t.Run("NotFound", func(t *testing.T) {
		storage, mock := newStorage(t)
		defer mock.Close()

		mock.ExpectQuery(regexp.QuoteMeta("FROM wallet_transaction")).
			WithArgs("absent").
			WillReturnError(pgx.ErrNoRows)

		_, err := storage.GetTransactionByIdempotencyKey(context.Background(), "absent")
		assert.ErrorIs(t, err, ErrTransactionNotFound)
	})
}

func TestWalletStorage_ListTransactionsByUser(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("FROM wallet_transaction")).
		WithArgs(int64(1), int64(0), 20).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "user_id", colAmount, "type", "reference_id", "idempotency_key", colCreatedAt,
			}).
				AddRow(int64(2), int64(1), int64(-49), models.WalletTxTypePromotionCharge, (*int64)(nil), (*string)(nil), now).
				AddRow(int64(1), int64(1), int64(500), models.WalletTxTypeTopup, (*int64)(nil), (*string)(nil), now),
		)

	items, err := storage.ListTransactionsByUser(context.Background(), 1, 0, 20)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, int64(2), items[0].ID)
	assert.Equal(t, int64(-49), items[0].Amount)
}

func TestWalletStorage_ListTransactionsByUser_Empty(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM wallet_transaction")).
		WithArgs(int64(1), int64(0), 20).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))

	items, err := storage.ListTransactionsByUser(context.Background(), 1, 0, 20)
	require.NoError(t, err)
	assert.NotNil(t, items)
	assert.Len(t, items, 0)
}

func TestWalletStorage_IncrementBalanceTx_Error(t *testing.T) {
	storage, mock := newStorage(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE wallet")).
		WithArgs(int64(1), int64(100)).
		WillReturnError(errors.New("constraint"))
	mock.ExpectRollback()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	_, err = storage.IncrementBalanceTx(context.Background(), tx, 1, 100)
	assert.Error(t, err)
	require.NoError(t, tx.Rollback(context.Background()))
}
