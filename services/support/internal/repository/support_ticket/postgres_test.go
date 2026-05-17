package supportticket

import (
	"context"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	colCreatedAt   = "created_at"
	colCategory    = "category"
	colCount       = "count"
	colRating      = "rating"
	colDescription = "description"
	colStatus      = "status"
	colTitle       = "title"
	colUpdatedAt   = "updated_at"
	colUserID      = "user_id"
)

func TestSupportTicketStorage_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	ticket := &models.SupportTicket{UserID: 7, Category: "bug", Title: "Test", Description: "Desc"}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO support_ticket")).
		WithArgs(ticket.UserID, ticket.Category, ticket.Title, ticket.Description).
		WillReturnRows(pgxmock.NewRows([]string{"id", colStatus, colRating, colCreatedAt, colUpdatedAt}).
			AddRow(int64(12), "open", nil, now, now))

	id, err := storage.Create(ctx, ticket)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), id)
	assert.Equal(t, int64(12), ticket.ID)
	assert.Equal(t, "open", ticket.Status)

	t.Run("UserNotFound", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23503"}
		mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO support_ticket")).
			WithArgs(ticket.UserID, ticket.Category, ticket.Title, ticket.Description).
			WillReturnError(pgErr)

		_, err := storage.Create(ctx, ticket)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestSupportTicketStorage_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	ticketID := int64(8)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", colUserID, colCategory, colStatus, colTitle, colDescription, colRating, colCreatedAt, colUpdatedAt}).
		AddRow(ticketID, int64(7), "bug", "open", "Title", "Desc", nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, category, status, title, description, rating, created_at, updated_at")).
		WithArgs(ticketID).
		WillReturnRows(rows)

	ticket, err := storage.GetByID(ctx, ticketID)
	assert.NoError(t, err)
	assert.Equal(t, ticketID, ticket.ID)
	assert.Equal(t, "bug", ticket.Category)

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, category, status, title, description, rating, created_at, updated_at")).
			WithArgs(ticketID).
			WillReturnError(pgx.ErrNoRows)

		_, err := storage.GetByID(ctx, ticketID)
		assert.ErrorIs(t, err, ErrTicketNotFound)
	})
}

func TestSupportTicketStorage_GetByUserID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	userID := int64(7)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", colUserID, colCategory, colStatus, colTitle, colDescription, colRating, colCreatedAt, colUpdatedAt}).
		AddRow(int64(1), userID, "bug", "open", "Title", "Desc", nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, category, status, title, description, rating, created_at, updated_at")).
		WithArgs(userID).
		WillReturnRows(rows)

	tickets, err := storage.GetByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, tickets, 1)
	assert.Equal(t, userID, tickets[0].UserID)

	t.Run("Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, category, status, title, description, rating, created_at, updated_at")).
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"id", colUserID, colCategory, colStatus, colTitle, colDescription, colRating, colCreatedAt, colUpdatedAt}))

		tickets, err := storage.GetByUserID(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, tickets)
		assert.Len(t, tickets, 0)
	})
}

func TestSupportTicketStorage_GetAll(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", colUserID, colCategory, colStatus, colTitle, colDescription, colRating, colCreatedAt, colUpdatedAt}).
		AddRow(int64(2), int64(8), "suggestion", "open", "Title", "Desc", nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, category, status, title, description, rating, created_at, updated_at")).
		WillReturnRows(rows)

	tickets, err := storage.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, tickets, 1)
	assert.Equal(t, int64(8), tickets[0].UserID)

	t.Run("Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, category, status, title, description, rating, created_at, updated_at")).
			WillReturnRows(pgxmock.NewRows([]string{"id", colUserID, colCategory, colStatus, colTitle, colDescription, colRating, colCreatedAt, colUpdatedAt}))

		tickets, err := storage.GetAll(ctx)
		assert.NoError(t, err)
		assert.Len(t, tickets, 0)
	})
}

func TestSupportTicketStorage_UpdateStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	updatedAt := time.Now()
	ticketID := int64(13)

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE support_ticket")).
		WithArgs("in_progress", ticketID).
		WillReturnRows(pgxmock.NewRows([]string{colUpdatedAt}).AddRow(updatedAt))

	got, err := storage.UpdateStatus(ctx, ticketID, "in_progress")
	assert.NoError(t, err)
	assert.WithinDuration(t, updatedAt, got, time.Second)

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("UPDATE support_ticket")).
			WithArgs("closed", ticketID).
			WillReturnError(pgx.ErrNoRows)

		_, err := storage.UpdateStatus(ctx, ticketID, "closed")
		assert.ErrorIs(t, err, ErrTicketNotFound)
	})
}

func TestSupportTicketStorage_GetStats(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM support_ticket")).
		WillReturnRows(pgxmock.NewRows([]string{colCount}).AddRow(int(5)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, COUNT(*) FROM support_ticket GROUP BY status")).
		WillReturnRows(pgxmock.NewRows([]string{colStatus, colCount}).AddRow("open", int(3)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT category, COUNT(*) FROM support_ticket GROUP BY category")).
		WillReturnRows(pgxmock.NewRows([]string{colCategory, colCount}).AddRow("bug", int(2)))

	stats, err := storage.GetStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 5, stats.Total)
	assert.Equal(t, 3, stats.ByStatus["open"])
	assert.Equal(t, 2, stats.ByCategory["bug"])
}

func TestSupportTicketStorage_SetRating(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	ticketID := int64(20)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE support_ticket")).
		WithArgs(5, ticketID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = storage.SetRating(ctx, ticketID, 5)
	assert.NoError(t, err)

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta("UPDATE support_ticket")).
			WithArgs(5, ticketID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := storage.SetRating(ctx, ticketID, 5)
		assert.ErrorIs(t, err, ErrTicketNotFound)
	})
}

func TestSupportTicketStorage_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	storage := NewSupportTicketStorage(mock, slog.Default())
	ctx := context.Background()
	ticket := &models.SupportTicket{ID: 30, UserID: 8, Category: "bug", Title: "Old", Description: "Old desc"}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE support_ticket")).
		WithArgs(ticket.Category, ticket.Title, ticket.Description, ticket.ID, ticket.UserID).
		WillReturnRows(pgxmock.NewRows([]string{colUpdatedAt}).AddRow(time.Now()))

	err = storage.Update(ctx, ticket)
	assert.NoError(t, err)

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("UPDATE support_ticket")).
			WithArgs(ticket.Category, ticket.Title, ticket.Description, ticket.ID, ticket.UserID).
			WillReturnError(pgx.ErrNoRows)

		err := storage.Update(ctx, ticket)
		assert.ErrorIs(t, err, ErrTicketNotFound)
	})
}
