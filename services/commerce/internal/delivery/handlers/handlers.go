// Package handlers — HTTP-обработчики commerce-сервиса.
package handlers

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/domain/dto"
)

// Общие константы ошибок. Эндпоинт-специфичные (chat-only / cart-only)
// объявлены в chat.go и cart.go, чтобы не дублировать.
const (
	ErrInvalidRequestBody = "invalid request body"
	ErrInternalError      = "internal error"
	ErrMethodNotAllowed   = "Method not allowed"
	ErrUnauthorized       = "unauthorized"
	ErrInvalidAdID        = "invalid ad id"
	ErrInvalidChatID      = "invalid chat id"
	ErrInvalidProductID   = "invalid product id"
	ErrForbidden          = "forbidden"
)

// Cart — usecase корзины.
type Cart interface {
	AddToCart(ctx context.Context, userID, productID int64) error
	RemoveFromCart(ctx context.Context, userID, productID int64) error
	GetCart(ctx context.Context, userID int64) (*dto.CartResponse, error)
}

// Chat — usecase чатов и заказов.
type Chat interface {
	CreateOrderRequest(ctx context.Context, adID int64, buyerID int64) (int64, error)
	ConfirmPurchase(ctx context.Context, chatID int64, userID int64) error
	GetAllChats(ctx context.Context, userID int64) (dto.ChatListResponse, error)
	GetChat(ctx context.Context, chatID, userID int64) (dto.ChatDetailResponse, error)
}

// Services — агрегатор зависимостей хендлеров. Cart/Chat-поля сохранены для
// совместимости с портированным из монолита кодом (h.services.Cart.* / h.services.Chat.*).
type Services struct {
	Cart Cart
	Chat Chat
}

// CartHandlers содержит обработчики корзины.
type CartHandlers struct {
	log      *slog.Logger
	services Services
}

// NewCartHandlers создаёт CartHandlers.
func NewCartHandlers(log *slog.Logger, cart Cart) *CartHandlers {
	return &CartHandlers{log: log, services: Services{Cart: cart}}
}

// ChatHandlers содержит обработчики чатов и заказов.
type ChatHandlers struct {
	log      *slog.Logger
	services *Services
}

// NewChatHandlers создаёт ChatHandlers.
func NewChatHandlers(log *slog.Logger, chat Chat) *ChatHandlers {
	return &ChatHandlers{log: log, services: &Services{Chat: chat}}
}
