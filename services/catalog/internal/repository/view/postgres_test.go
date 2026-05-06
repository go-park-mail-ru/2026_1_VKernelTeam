package view

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) (*ViewStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return NewViewStorage(mock, slog.Default()), mock
}

// ─── BatchInsertViews ────────────────────────────────────────────────────────

func TestBatchInsertViews_EmptySlice(t *testing.T) {
	storage, mock := newTestStorage(t)

	err := storage.BatchInsertViews(context.Background(), nil)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_SingleEvent(t *testing.T) {
	storage, mock := newTestStorage(t)
	ctx := context.Background()
	now := time.Now()
	userID := int64(42)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO product_view (product_id, user_id, viewed_at) VALUES ($1, $2, $3)",
	)).WithArgs(int64(1), &userID, now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE product SET views_count = views_count + $1 WHERE id = $2",
	)).WithArgs(int64(1), int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	err := storage.BatchInsertViews(ctx, []InsertEvent{
		{ProductID: 1, UserID: &userID, ViewedAt: now},
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_NilUserID(t *testing.T) {
	storage, mock := newTestStorage(t)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO product_view (product_id, user_id, viewed_at) VALUES ($1, $2, $3)",
	)).WithArgs(int64(5), (*int64)(nil), now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE product SET views_count = views_count + $1 WHERE id = $2",
	)).WithArgs(int64(1), int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	err := storage.BatchInsertViews(ctx, []InsertEvent{
		{ProductID: 5, UserID: nil, ViewedAt: now},
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_MultipleEventsMultipleProducts(t *testing.T) {
	storage, mock := newTestStorage(t)
	ctx := context.Background()
	now := time.Now()
	uid1 := int64(10)
	uid2 := int64(20)

	events := []InsertEvent{
		{ProductID: 1, UserID: &uid1, ViewedAt: now},
		{ProductID: 1, UserID: &uid2, ViewedAt: now},
		{ProductID: 2, UserID: (*int64)(nil), ViewedAt: now},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO product_view (product_id, user_id, viewed_at) VALUES ($1, $2, $3), ($4, $5, $6), ($7, $8, $9)",
	)).WithArgs(
		int64(1), &uid1, now,
		int64(1), &uid2, now,
		int64(2), (*int64)(nil), now,
	).WillReturnResult(pgxmock.NewResult("INSERT", 3))

	// map iteration order is non-deterministic, so accept any args
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE product SET views_count = views_count + $1 WHERE id = $2",
	)).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE product SET views_count = views_count + $1 WHERE id = $2",
	)).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectCommit()

	err := storage.BatchInsertViews(ctx, events)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_BeginError(t *testing.T) {
	storage, mock := newTestStorage(t)

	mock.ExpectBegin().WillReturnError(fmt.Errorf("connection refused"))

	err := storage.BatchInsertViews(context.Background(), []InsertEvent{
		{ProductID: 1, ViewedAt: time.Now()},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin tx")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_InsertError(t *testing.T) {
	storage, mock := newTestStorage(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO product_view",
	)).WithArgs(int64(1), (*int64)(nil), now).
		WillReturnError(fmt.Errorf("unique violation"))
	mock.ExpectRollback()

	err := storage.BatchInsertViews(context.Background(), []InsertEvent{
		{ProductID: 1, ViewedAt: now},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_UpdateCountError(t *testing.T) {
	storage, mock := newTestStorage(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO product_view",
	)).WithArgs(int64(1), (*int64)(nil), now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE product SET views_count",
	)).WithArgs(int64(1), int64(1)).
		WillReturnError(fmt.Errorf("fk violation"))
	mock.ExpectRollback()

	err := storage.BatchInsertViews(context.Background(), []InsertEvent{
		{ProductID: 1, ViewedAt: now},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update count")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchInsertViews_CommitError(t *testing.T) {
	storage, mock := newTestStorage(t)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO product_view",
	)).WithArgs(int64(1), (*int64)(nil), now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE product SET views_count",
	)).WithArgs(int64(1), int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("serialization failure"))
	mock.ExpectRollback()

	err := storage.BatchInsertViews(context.Background(), []InsertEvent{
		{ProductID: 1, ViewedAt: now},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─── GetViewsCount ───────────────────────────────────────────────────────────

func TestGetViewsCount_Success(t *testing.T) {
	storage, mock := newTestStorage(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT views_count FROM product WHERE id = $1 AND deleted_at IS NULL",
	)).WithArgs(int64(7)).
		WillReturnRows(pgxmock.NewRows([]string{"views_count"}).AddRow(int64(150)))

	count, err := storage.GetViewsCount(context.Background(), 7)

	assert.NoError(t, err)
	assert.Equal(t, int64(150), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetViewsCount_ZeroViews(t *testing.T) {
	storage, mock := newTestStorage(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT views_count FROM product WHERE id = $1 AND deleted_at IS NULL",
	)).WithArgs(int64(3)).
		WillReturnRows(pgxmock.NewRows([]string{"views_count"}).AddRow(int64(0)))

	count, err := storage.GetViewsCount(context.Background(), 3)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetViewsCount_NotFound(t *testing.T) {
	storage, mock := newTestStorage(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT views_count FROM product WHERE id = $1 AND deleted_at IS NULL",
	)).WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	count, err := storage.GetViewsCount(context.Background(), 999)

	assert.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetViewsCount_DBError(t *testing.T) {
	storage, mock := newTestStorage(t)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT views_count FROM product WHERE id = $1 AND deleted_at IS NULL",
	)).WithArgs(int64(1)).
		WillReturnError(fmt.Errorf("connection lost"))

	count, err := storage.GetViewsCount(context.Background(), 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
