package promotion

import (
	"context"
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

func newPromotionStorage(t *testing.T) (*PromotionStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	return NewPromotionStorage(mock, slog.Default()), mock
}

func promotionRow(id, productID, userID, planID int64, kind string, expires time.Time) []any {
	now := time.Now()
	return []any{id, productID, userID, planID, kind, now, expires, int64(199), now}
}

func promotionCols() []string {
	return []string{
		"id", "product_id", "user_id", "plan_id", "kind",
		"starts_at", "expires_at", "price_paid", "created_at",
	}
}

func TestPromotionStorage_InsertTx_Success(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	expires := time.Now().Add(7 * 24 * time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO promotion")).
		WithArgs(int64(10), int64(1), int64(7), models.PromotionKindBoost, expires, int64(199)).
		WillReturnRows(pgxmock.NewRows(promotionCols()).AddRow(
			promotionRow(555, 10, 1, 7, models.PromotionKindBoost, expires)...,
		))
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	promo, err := storage.InsertTx(context.Background(), tx, 10, 1, 7, models.PromotionKindBoost, expires, 199)
	require.NoError(t, err)
	assert.Equal(t, int64(555), promo.ID)
	assert.Equal(t, models.PromotionKindBoost, promo.Kind)

	require.NoError(t, tx.Commit(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPromotionStorage_MaxActiveExpiresTx_Has(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	want := time.Now().Add(3 * 24 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT MAX(expires_at)")).
		WithArgs(int64(10), models.PromotionKindBoost).
		WillReturnRows(pgxmock.NewRows([]string{"max"}).AddRow(&want))
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	got, has, err := storage.MaxActiveExpiresTx(context.Background(), tx, 10, models.PromotionKindBoost)
	require.NoError(t, err)
	assert.True(t, has)
	assert.WithinDuration(t, want, got, time.Second)

	require.NoError(t, tx.Commit(context.Background()))
}

func TestPromotionStorage_MaxActiveExpiresTx_None(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT MAX(expires_at)")).
		WithArgs(int64(10), models.PromotionKindBoost).
		WillReturnRows(pgxmock.NewRows([]string{"max"}).AddRow((*time.Time)(nil)))
	mock.ExpectCommit()

	tx, err := mock.Begin(context.Background())
	require.NoError(t, err)

	_, has, err := storage.MaxActiveExpiresTx(context.Background(), tx, 10, models.PromotionKindBoost)
	require.NoError(t, err)
	assert.False(t, has)

	require.NoError(t, tx.Commit(context.Background()))
}

func TestPromotionStorage_GetActiveByAd(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	expires := time.Now().Add(time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE product_id = $1 AND expires_at >")).
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows(promotionCols()).
			AddRow(promotionRow(1, 10, 1, 7, models.PromotionKindBoost, expires)...).
			AddRow(promotionRow(2, 10, 1, 8, models.PromotionKindHighlight, expires)...),
		)

	promos, err := storage.GetActiveByAd(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, promos, 2)
}

func TestPromotionStorage_GetActiveByAd_Empty(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE product_id")).
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))

	promos, err := storage.GetActiveByAd(context.Background(), 10)
	require.NoError(t, err)
	assert.NotNil(t, promos)
	assert.Len(t, promos, 0)
}

func TestPromotionStorage_ListByUser(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	expires := time.Now().Add(time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE user_id = $1")).
		WithArgs(int64(1), int64(0), 20).
		WillReturnRows(pgxmock.NewRows(promotionCols()).
			AddRow(promotionRow(5, 10, 1, 7, models.PromotionKindBoost, expires)...),
		)

	promos, err := storage.ListByUser(context.Background(), 1, 0, 20)
	require.NoError(t, err)
	require.Len(t, promos, 1)
	assert.Equal(t, int64(5), promos[0].ID)
}

func TestPromotionStorage_GetByID_Success(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	expires := time.Now().Add(time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1")).
		WithArgs(int64(555)).
		WillReturnRows(pgxmock.NewRows(promotionCols()).
			AddRow(promotionRow(555, 10, 1, 7, models.PromotionKindBoost, expires)...),
		)

	promo, err := storage.GetByID(context.Background(), 555)
	require.NoError(t, err)
	assert.Equal(t, int64(555), promo.ID)
}

func TestPromotionStorage_GetByID_NotFound(t *testing.T) {
	storage, mock := newPromotionStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1")).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err := storage.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, ErrPromotionNotFound)
}
