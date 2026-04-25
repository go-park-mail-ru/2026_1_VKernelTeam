package supportticket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	opCreate       = "db.support_ticket.Create"
	opGetByID      = "db.support_ticket.GetByID"
	opGetByUserID  = "db.support_ticket.GetByUserID"
	opUpdate       = "db.support_ticket.Update"
	opGetAll       = "db.support_ticket.GetAll"
	opUpdateStatus = "db.support_ticket.UpdateStatus"
	opGetStats     = "db.support_ticket.GetStats"
)

type PgxPool interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
}

var (
	ErrTicketNotFound = errors.New("ticket not found")
)

type SupportTicketStorage struct {
	pool PgxPool
	log  *slog.Logger
}

func NewSupportTicketStorage(pool PgxPool, log *slog.Logger) *SupportTicketStorage {
	return &SupportTicketStorage{pool: pool, log: log}
}

func (s *SupportTicketStorage) Create(ctx context.Context, ticket *models.SupportTicket) (int64, error) {
	const query = `
		INSERT INTO support_ticket (user_id, category, title, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, created_at, updated_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opCreate),
		slog.Int64("user_id", ticket.UserID),
	)

	err := s.pool.QueryRow(ctx, query,
		ticket.UserID, ticket.Category, ticket.Title, ticket.Description,
	).Scan(&ticket.ID, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create ticket",
			slog.String("op", opCreate),
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("SupportTicketStorage.Create: %w", err)
	}

	return ticket.ID, nil
}

func (s *SupportTicketStorage) GetByID(ctx context.Context, id int64) (*models.SupportTicket, error) {
	const query = `
		SELECT id, user_id, category, status, title, description, created_at, updated_at
		FROM support_ticket
		WHERE id = $1
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetByID),
		slog.Int64("id", id),
	)

	var t models.SupportTicket
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.UserID, &t.Category, &t.Status,
		&t.Title, &t.Description, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		s.log.ErrorContext(ctx, "failed to get ticket",
			slog.String("op", opGetByID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportTicketStorage.GetByID: %w", err)
	}

	return &t, nil
}

func (s *SupportTicketStorage) GetByUserID(ctx context.Context, userID int64) ([]models.SupportTicket, error) {
	const query = `
		SELECT id, user_id, category, status, title, description, created_at, updated_at
		FROM support_ticket
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opGetByUserID),
		slog.Int64("user_id", userID),
	)

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get tickets by user",
			slog.String("op", opGetByUserID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportTicketStorage.GetByUserID: %w", err)
	}
	defer rows.Close()

	var tickets []models.SupportTicket
	for rows.Next() {
		var t models.SupportTicket
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Category, &t.Status,
			&t.Title, &t.Description, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			s.log.ErrorContext(ctx, "failed to scan ticket",
				slog.String("op", opGetByUserID),
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("SupportTicketStorage.GetByUserID: scan: %w", err)
		}
		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SupportTicketStorage.GetByUserID: rows: %w", err)
	}

	if tickets == nil {
		tickets = []models.SupportTicket{}
	}

	return tickets, nil
}

// GetAll возвращает все обращения всех пользователей, отсортированные по дате создания.
func (s *SupportTicketStorage) GetAll(ctx context.Context) ([]models.SupportTicket, error) {
	const query = `
		SELECT id, user_id, category, status, title, description, created_at, updated_at
		FROM support_ticket
		ORDER BY created_at DESC
	`

	s.log.DebugContext(ctx, "executing query", slog.String("op", opGetAll))

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get all tickets",
			slog.String("op", opGetAll),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportTicketStorage.GetAll: %w", err)
	}
	defer rows.Close()

	tickets := []models.SupportTicket{}
	for rows.Next() {
		var t models.SupportTicket
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Category, &t.Status,
			&t.Title, &t.Description, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("SupportTicketStorage.GetAll: scan: %w", err)
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SupportTicketStorage.GetAll: rows: %w", err)
	}

	return tickets, nil
}

// UpdateStatus меняет статус обращения и возвращает обновлённое значение updated_at.
func (s *SupportTicketStorage) UpdateStatus(ctx context.Context, ticketID int64, status string) (time.Time, error) {
	const query = `
		UPDATE support_ticket
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opUpdateStatus),
		slog.Int64("ticket_id", ticketID),
		slog.String("status", status),
	)

	var updatedAt time.Time
	err := s.pool.QueryRow(ctx, query, status, ticketID).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrTicketNotFound
		}
		s.log.ErrorContext(ctx, "failed to update ticket status",
			slog.String("op", opUpdateStatus),
			slog.String("error", err.Error()),
		)
		return time.Time{}, fmt.Errorf("SupportTicketStorage.UpdateStatus: %w", err)
	}

	return updatedAt, nil
}

// GetStats возвращает сводную статистику по обращениям: total, разбивка по статусу и категории.
func (s *SupportTicketStorage) GetStats(ctx context.Context) (*dto.StatsResponse, error) {
	s.log.DebugContext(ctx, "executing query", slog.String("op", opGetStats))

	stats := &dto.StatsResponse{
		ByStatus:   map[string]int{},
		ByCategory: map[string]int{},
	}

	const totalQuery = `SELECT COUNT(*) FROM support_ticket`
	if err := s.pool.QueryRow(ctx, totalQuery).Scan(&stats.Total); err != nil {
		s.log.ErrorContext(ctx, "failed to get total tickets",
			slog.String("op", opGetStats),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportTicketStorage.GetStats: total: %w", err)
	}

	const byStatusQuery = `SELECT status, COUNT(*) FROM support_ticket GROUP BY status`
	if err := scanCounts(ctx, s.pool, byStatusQuery, stats.ByStatus); err != nil {
		s.log.ErrorContext(ctx, "failed to get tickets by status",
			slog.String("op", opGetStats),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportTicketStorage.GetStats: by_status: %w", err)
	}

	const byCategoryQuery = `SELECT category, COUNT(*) FROM support_ticket GROUP BY category`
	if err := scanCounts(ctx, s.pool, byCategoryQuery, stats.ByCategory); err != nil {
		s.log.ErrorContext(ctx, "failed to get tickets by category",
			slog.String("op", opGetStats),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("SupportTicketStorage.GetStats: by_category: %w", err)
	}

	return stats, nil
}

func scanCounts(ctx context.Context, pool PgxPool, query string, dst map[string]int) error {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return err
		}
		dst[key] = count
	}
	return rows.Err()
}

func (s *SupportTicketStorage) Update(ctx context.Context, ticket *models.SupportTicket) error {
	const query = `
		UPDATE support_ticket
		SET category = $1, title = $2, description = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING updated_at
	`

	s.log.DebugContext(ctx, "executing query",
		slog.String("op", opUpdate),
		slog.Int64("id", ticket.ID),
		slog.Int64("user_id", ticket.UserID),
	)

	err := s.pool.QueryRow(ctx, query,
		ticket.Category, ticket.Title, ticket.Description,
		ticket.ID, ticket.UserID,
	).Scan(&ticket.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTicketNotFound
		}
		s.log.ErrorContext(ctx, "failed to update ticket",
			slog.String("op", opUpdate),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("SupportTicketStorage.Update: %w", err)
	}

	return nil
}
