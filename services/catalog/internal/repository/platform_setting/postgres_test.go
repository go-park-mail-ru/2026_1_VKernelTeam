package platform_setting

import (
	"context"
	"log/slog"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

func TestStorage_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	s := New(mock, slog.Default())
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM platform_setting")).
			WithArgs("k").
			WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow("v"))
		v, err := s.Get(ctx, "k")
		assert.NoError(t, err)
		assert.Equal(t, "v", v)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM platform_setting")).
			WithArgs("k").
			WillReturnError(pgx.ErrNoRows)
		_, err := s.Get(ctx, "k")
		assert.ErrorIs(t, err, ErrSettingNotFound)
	})

	t.Run("other error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM platform_setting")).
			WithArgs("k").
			WillReturnError(assert.AnError)
		_, err := s.Get(ctx, "k")
		assert.Error(t, err)
	})
}

func TestStorage_Set(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	s := New(mock, slog.Default())
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO platform_setting")).
			WithArgs("k", "v", int64(7)).
			WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))
		assert.NoError(t, s.Set(ctx, "k", "v", 7))
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO platform_setting")).
			WithArgs("k", "v", int64(7)).
			WillReturnError(assert.AnError)
		assert.Error(t, s.Set(ctx, "k", "v", 7))
	})
}
