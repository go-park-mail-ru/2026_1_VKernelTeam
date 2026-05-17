package supportmessage

import (
	"context"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/models"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const colCreatedAt = "created_at"

func TestSupportMessageStorage_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportMessageStorage(mock, slog.Default())
	ctx := context.Background()
	msg := &models.SupportMessage{TicketID: 1, UserID: 2, Text: "Hello"}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO support_message")).
		WithArgs(msg.TicketID, msg.UserID, msg.Text).
		WillReturnRows(pgxmock.NewRows([]string{"id", colCreatedAt}).AddRow(int64(10), now))

	id, err := storage.Create(ctx, msg)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), id)
	assert.Equal(t, int64(10), msg.ID)
	assert.WithinDuration(t, now, msg.CreatedAt, time.Second)

	require.NoError(t, mock.ExpectationsWereMet())

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO support_message")).
			WithArgs(msg.TicketID, msg.UserID, msg.Text).
			WillReturnError(assert.AnError)

		_, err := storage.Create(ctx, msg)
		assert.Error(t, err)
	})
}

func TestSupportMessageStorage_GetByTicketID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportMessageStorage(mock, slog.Default())
	ctx := context.Background()
	ticketID := int64(5)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "ticket_id", "user_id", "text", colCreatedAt}).
		AddRow(int64(1), ticketID, int64(2), "first", now).
		AddRow(int64(2), ticketID, int64(3), "second", now.Add(time.Minute))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, ticket_id, user_id, text, created_at")).
		WithArgs(ticketID).
		WillReturnRows(rows)

	messages, err := storage.GetByTicketID(ctx, ticketID)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "first", messages[0].Text)
	assert.Equal(t, "second", messages[1].Text)

	t.Run("EmptyResult", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, ticket_id, user_id, text, created_at")).
			WithArgs(ticketID).
			WillReturnRows(pgxmock.NewRows([]string{"id", "ticket_id", "user_id", "text", colCreatedAt}))

		messages, err := storage.GetByTicketID(ctx, ticketID)
		assert.NoError(t, err)
		assert.Len(t, messages, 0)
	})

	t.Run("QueryError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, ticket_id, user_id, text, created_at")).
			WithArgs(ticketID).
			WillReturnError(assert.AnError)

		_, err := storage.GetByTicketID(ctx, ticketID)
		assert.Error(t, err)
	})
}
