package supportticket

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

var (
	ErrInvalidCategory  = errors.New("invalid category: must be bug, suggestion or complaint")
	ErrTitleTooLong     = errors.New("title must be at most 255 characters")
	ErrTitleRequired    = errors.New("title is required")
	ErrDescRequired     = errors.New("description is required")
	ErrCategoryRequired = errors.New("category is required")
	ErrTicketNotOpen    = errors.New("can only update tickets with status open")
	ErrForbidden        = errors.New("forbidden: not the ticket author")
)

var allowedCategories = map[string]bool{
	"bug":        true,
	"suggestion": true,
	"complaint":  true,
}

type TicketStorage interface {
	Create(ctx context.Context, ticket *models.SupportTicket) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.SupportTicket, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.SupportTicket, error)
	Update(ctx context.Context, ticket *models.SupportTicket) error
}

type SupportTicketService struct {
	log     *slog.Logger
	storage TicketStorage
}

func New(log *slog.Logger, storage TicketStorage) *SupportTicketService {
	return &SupportTicketService{log: log, storage: storage}
}

func (s *SupportTicketService) CreateTicket(ctx context.Context, userID int64, req *dto.CreateTicketRequest) (*dto.TicketResponse, error) {
	if err := validateTicketInput(req.Category, req.Title, req.Description); err != nil {
		return nil, err
	}

	ticket := &models.SupportTicket{
		UserID:      userID,
		Category:    req.Category,
		Title:       req.Title,
		Description: req.Description,
	}

	_, err := s.storage.Create(ctx, ticket)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to create ticket", slog.String("error", err.Error()))
		return nil, err
	}

	return toTicketResponse(ticket), nil
}

func (s *SupportTicketService) GetMyTickets(ctx context.Context, userID int64) ([]dto.TicketResponse, error) {
	tickets, err := s.storage.GetByUserID(ctx, userID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get user tickets", slog.String("error", err.Error()))
		return nil, err
	}

	result := make([]dto.TicketResponse, len(tickets))
	for i, t := range tickets {
		result[i] = *toTicketResponse(&t)
	}
	return result, nil
}

func (s *SupportTicketService) GetTicket(ctx context.Context, ticketID, userID int64) (*dto.TicketResponse, error) {
	ticket, err := s.storage.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	// Доступ: автор тикета (роль support/admin — будущее расширение)
	if ticket.UserID != userID {
		return nil, ErrForbidden
	}

	return toTicketResponse(ticket), nil
}

func (s *SupportTicketService) UpdateTicket(ctx context.Context, ticketID, userID int64, req *dto.UpdateTicketRequest) (*dto.TicketResponse, error) {
	ticket, err := s.storage.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if ticket.UserID != userID {
		return nil, ErrForbidden
	}

	if ticket.Status != "open" {
		return nil, ErrTicketNotOpen
	}

	if err := validateTicketInput(req.Category, req.Title, req.Description); err != nil {
		return nil, err
	}

	ticket.Category = req.Category
	ticket.Title = req.Title
	ticket.Description = req.Description

	if err := s.storage.Update(ctx, ticket); err != nil {
		s.log.ErrorContext(ctx, "failed to update ticket", slog.String("error", err.Error()))
		return nil, err
	}

	return toTicketResponse(ticket), nil
}

func validateTicketInput(category, title, description string) error {
	if category == "" {
		return ErrCategoryRequired
	}
	if !allowedCategories[category] {
		return ErrInvalidCategory
	}
	if title == "" {
		return ErrTitleRequired
	}
	if len(title) > 255 {
		return ErrTitleTooLong
	}
	if description == "" {
		return ErrDescRequired
	}
	return nil
}

func toTicketResponse(t *models.SupportTicket) *dto.TicketResponse {
	return &dto.TicketResponse{
		ID:          t.ID,
		UserID:      t.UserID,
		Category:    t.Category,
		Status:      t.Status,
		Title:       t.Title,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
