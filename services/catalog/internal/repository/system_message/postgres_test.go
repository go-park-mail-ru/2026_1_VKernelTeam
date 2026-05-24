package system_message

import (
	"context"
	"log/slog"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
)

func TestStorage_Send(t *testing.T) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	s := New(mock, slog.Default())
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO chat")).
			WithArgs(int64(10), int64(1), int64(99)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(5)))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO message")).
			WithArgs(int64(5), int64(1), "hi", messageTypeSystem).
			WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

		assert.NoError(t, s.Send(ctx, 1, 99, 10, "hi"))
	})

	t.Run("ad_id required", func(t *testing.T) {
		assert.Error(t, s.Send(ctx, 1, 99, 0, "hi"))
	})

	t.Run("chat upsert error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO chat")).
			WithArgs(int64(10), int64(1), int64(99)).
			WillReturnError(assert.AnError)
		assert.Error(t, s.Send(ctx, 1, 99, 10, "hi"))
	})

	t.Run("message insert error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO chat")).
			WithArgs(int64(10), int64(1), int64(99)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(5)))
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO message")).
			WithArgs(int64(5), int64(1), "hi", messageTypeSystem).
			WillReturnError(assert.AnError)
		assert.Error(t, s.Send(ctx, 1, 99, 10, "hi"))
	})
}
