package handlers

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/domain/dto"
)

const (
	ErrInvalidRequestBody = "invalid request body"
	ErrUnauthorized       = "unauthorized"
	ErrForbidden          = "forbidden"
	ErrInternalError      = "internal error"
)

// SupportTicket описывает методы сервиса тикетов техподдержки
type SupportTicket interface {
	CreateTicket(ctx context.Context, userID int64, req *dto.CreateTicketRequest) (*dto.TicketResponse, error)
	GetMyTickets(ctx context.Context, userID int64) ([]dto.TicketResponse, error)
	GetTicket(ctx context.Context, ticketID, userID int64) (*dto.TicketResponse, error)
	UpdateTicket(ctx context.Context, ticketID, userID int64, req *dto.UpdateTicketRequest) (*dto.TicketResponse, error)
	GetAllTickets(ctx context.Context) ([]dto.TicketResponse, error)
	ChangeStatus(ctx context.Context, ticketID int64, req *dto.ChangeStatusRequest) (*dto.TicketStatusResponse, error)
	GetStats(ctx context.Context) (*dto.StatsResponse, error)
	RateTicket(ctx context.Context, userID, ticketID int64, rating int) (*dto.TicketResponse, error)
}

// SupportMessage описывает методы сервиса сообщений обращения
type SupportMessage interface {
	SendMessage(ctx context.Context, ticketID, userID int64, req *dto.SendMessageRequest) (*dto.MessageResponse, error)
	GetMessages(ctx context.Context, ticketID, userID int64) ([]dto.MessageResponse, error)
}

// Services агрегирует сервисы техподдержки
type Services struct {
	SupportTicket  SupportTicket
	SupportMessage SupportMessage
}

// SupportTicketHandlers обрабатывает запросы техподдержки
type SupportTicketHandlers struct {
	log      *slog.Logger
	services *Services
}

// NewSupportTicketHandlers создает новый экземпляр SupportTicketHandlers
func NewSupportTicketHandlers(log *slog.Logger, services *Services) *SupportTicketHandlers {
	return &SupportTicketHandlers{
		log:      log,
		services: services,
	}
}
