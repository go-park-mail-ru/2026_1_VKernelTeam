package supportmessage

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

//go:generate mockgen -source=support_message.go -destination=mocks/mock_support_message.go -package=mocks

var (
	ErrTextRequired = errors.New("text is required")
	ErrForbidden    = errors.New("forbidden: not the ticket author or staff")
)

const (
	roleSupport = "support"
	roleAdmin   = "admin"
)

// TicketStorage используется для проверки автора тикета.
type TicketStorage interface {
	GetByID(ctx context.Context, id int64) (*models.SupportTicket, error)
}

// MessageStorage отвечает за хранение сообщений обращений.
type MessageStorage interface {
	Create(ctx context.Context, msg *models.SupportMessage) (int64, error)
	GetByTicketID(ctx context.Context, ticketID int64) ([]models.SupportMessage, error)
}

// RoleProvider возвращает роль пользователя по его ID.
type RoleProvider interface {
	GetUserRole(ctx context.Context, userID int64) (string, error)
}

type SupportMessageService struct {
	log      *slog.Logger
	messages MessageStorage
	tickets  TicketStorage
	users    RoleProvider
}

func New(log *slog.Logger, messages MessageStorage, tickets TicketStorage, users RoleProvider) *SupportMessageService {
	return &SupportMessageService{
		log:      log,
		messages: messages,
		tickets:  tickets,
		users:    users,
	}
}

// SendMessage отправляет сообщение в чат обращения.
// Доступ: автор тикета или пользователь с ролью support/admin.
func (s *SupportMessageService) SendMessage(
	ctx context.Context,
	ticketID, userID int64,
	req *dto.SendMessageRequest,
) (*dto.MessageResponse, error) {
	if req.Text == "" {
		return nil, ErrTextRequired
	}

	if err := s.checkAccess(ctx, ticketID, userID); err != nil {
		return nil, err
	}

	msg := &models.SupportMessage{
		TicketID: ticketID,
		UserID:   userID,
		Text:     req.Text,
	}

	if _, err := s.messages.Create(ctx, msg); err != nil {
		s.log.ErrorContext(ctx, "failed to create support message", slog.String("error", err.Error()))
		return nil, err
	}

	return toMessageResponse(msg), nil
}

// GetMessages возвращает все сообщения обращения.
// Доступ: автор тикета или пользователь с ролью support/admin.
func (s *SupportMessageService) GetMessages(
	ctx context.Context,
	ticketID, userID int64,
) ([]dto.MessageResponse, error) {
	if err := s.checkAccess(ctx, ticketID, userID); err != nil {
		return nil, err
	}

	messages, err := s.messages.GetByTicketID(ctx, ticketID)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get support messages", slog.String("error", err.Error()))
		return nil, err
	}

	result := make([]dto.MessageResponse, len(messages))
	for i, m := range messages {
		result[i] = *toMessageResponse(&m)
	}
	return result, nil
}

// checkAccess допускает к чату только автора тикета или сотрудников поддержки/админов.
func (s *SupportMessageService) checkAccess(ctx context.Context, ticketID, userID int64) error {
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}

	if ticket.UserID == userID {
		return nil
	}

	role, err := s.users.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}
	if role == roleSupport || role == roleAdmin {
		return nil
	}

	return ErrForbidden
}

func toMessageResponse(m *models.SupportMessage) *dto.MessageResponse {
	return &dto.MessageResponse{
		ID:        m.ID,
		TicketID:  m.TicketID,
		UserID:    m.UserID,
		Text:      m.Text,
		CreatedAt: m.CreatedAt,
	}
}
