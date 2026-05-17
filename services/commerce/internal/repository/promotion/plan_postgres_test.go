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

const colCode = "code"

func newPlanStorage(t *testing.T) (*PlanStorage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	return NewPlanStorage(mock, slog.Default()), mock
}

func planRow(now time.Time, isActive bool) []any {
	return []any{int64(7), "boost_7d", "boost", 7, int64(199), isActive, now, now}
}

func TestPlanStorage_GetActivePlans_Success(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("FROM promotion_plan")).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", colCode, colKind, "duration_days", "price", "is_active", colCreatedAt, "updated_at",
			}).
				AddRow(planRow(now, true)...).
				AddRow(int64(8), "highlight_7d", "highlight", 7, int64(99), true, now, now),
		)

	plans, err := storage.GetActivePlans(context.Background())
	require.NoError(t, err)
	require.Len(t, plans, 2)
	assert.Equal(t, "boost_7d", plans[0].Code)
	assert.Equal(t, models.PromotionKindHighlight, plans[1].Kind)
}

func TestPlanStorage_GetActivePlans_Empty(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("FROM promotion_plan")).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))

	plans, err := storage.GetActivePlans(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, plans)
	assert.Len(t, plans, 0)
}

func TestPlanStorage_GetByCode_Success(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE code = $1")).
		WithArgs("boost_7d").
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", colCode, colKind, "duration_days", "price", "is_active", colCreatedAt, "updated_at",
			}).AddRow(planRow(now, true)...),
		)

	plan, err := storage.GetByCode(context.Background(), "boost_7d")
	require.NoError(t, err)
	assert.Equal(t, int64(7), plan.ID)
	assert.True(t, plan.IsActive)
}

func TestPlanStorage_GetByCode_NotFound(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE code = $1")).
		WithArgs("ghost").
		WillReturnError(pgx.ErrNoRows)

	_, err := storage.GetByCode(context.Background(), "ghost")
	assert.ErrorIs(t, err, ErrPlanNotFound)
}

func TestPlanStorage_GetByCode_Inactive(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE code = $1")).
		WithArgs("boost_7d").
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", colCode, colKind, "duration_days", "price", "is_active", colCreatedAt, "updated_at",
			}).AddRow(planRow(now, false)...),
		)

	_, err := storage.GetByCode(context.Background(), "boost_7d")
	assert.ErrorIs(t, err, ErrPlanInactive)
}

func TestPlanStorage_GetByID_Success(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1")).
		WithArgs(int64(7)).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id", colCode, colKind, "duration_days", "price", "is_active", colCreatedAt, "updated_at",
			}).AddRow(planRow(now, true)...),
		)

	plan, err := storage.GetByID(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, "boost_7d", plan.Code)
}

func TestPlanStorage_GetByID_NotFound(t *testing.T) {
	storage, mock := newPlanStorage(t)
	defer mock.Close()

	mock.ExpectQuery(regexp.QuoteMeta("WHERE id = $1")).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err := storage.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, ErrPlanNotFound)
}
