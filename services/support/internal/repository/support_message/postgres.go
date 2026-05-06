package supportmessage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opCreate        = "db.support_message.Create"
	opGetByTicketID = "db.support_message.GetByTicketID"
)

// PgxPool интерфейс для пула соединений pgx.
type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

// SupportMessageStorage отвечает за операции с сообщениями обращений.
type SupportMessageStorage struct {
	pool PgxPool
	log  *slog.Logger
}

func NewSupportMessageStorage(pool PgxPool, log *slog.Logger) *SupportMessageStorage {
	return &SupportMessageStorage{pool: pool, log: log}
}

// Create сохраняет новое сообщение обращения.
func (s *SupportMessageStorage) Create(ctx context.Context, msg *models.SupportMessage) (int64, error) {
	const query = `
		INSERT INTO support_message (ticket_id, user_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCreate),
		slog.Int64("ticket_id", msg.TicketID),
		slog.Int64("user_id", msg.UserID),
	)

	err := s.pool.QueryRow(ctx, query, msg.TicketID, msg.UserID, msg.Text).
		Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create support message",
			slog.String("op", opCreate),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("SupportMessageStorage.Create: %w", err)
	}

	return msg.ID, nil
}

// GetByTicketID возвращает все сообщения обращения, отсортированные по времени.
func (s *SupportMessageStorage) GetByTicketID(ctx context.Context, ticketID int64) ([]models.SupportMessage, error) {
	const query = `
		SELECT id, ticket_id, user_id, text, created_at
		FROM support_message
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetByTicketID),
		slog.Int64("ticket_id", ticketID),
	)

	rows, err := s.pool.Query(ctx, query, ticketID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get messages by ticket",
			slog.String("op", opGetByTicketID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportMessageStorage.GetByTicketID: %w", err)
	}
	defer rows.Close()

	messages := []models.SupportMessage{}
	for rows.Next() {
		var m models.SupportMessage
		if err := rows.Scan(&m.ID, &m.TicketID, &m.UserID, &m.Text, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("SupportMessageStorage.GetByTicketID: scan: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SupportMessageStorage.GetByTicketID: rows: %w", err)
	}

	return messages, nil
}
