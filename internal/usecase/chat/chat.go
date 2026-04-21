package chat

//go:generate mockgen -source=chat.go -destination=mocks/mock_chat.go -package=mocks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

const (
	opCreateOrderRequest = "usecase.chat.CreateOrderRequest"
	opConfirmPurchase    = "usecase.chat.ConfirmPurchase"
)

// ChatProvider описывает интерфейс для работы с чатами в репозитории
type ChatProvider interface {
	GetOrCreateChat(ctx context.Context, adID int64, buyerID int64, sellerID int64) (int64, error)
	CreateMessage(ctx context.Context, message *models.Message) (int64, error)
	UpdateAdStatus(ctx context.Context, adID int64, status string) error
}

// AdProvider описывает интерфейс для получения данных объявления
type AdProvider interface {
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
}

// Chat описывает сервис для работы с чатами и заказами
type Chat struct {
	log         *slog.Logger
	chatStorage ChatProvider
	adStorage   AdProvider
}

// New создает новый экземпляр Chat с переданными зависимостями.
func New(
	log *slog.Logger,
	chatStorage ChatProvider,
	adStorage AdProvider,
) *Chat {
	return &Chat{
		log:         log,
		chatStorage: chatStorage,
		adStorage:   adStorage,
	}
}

// CreateOrderRequest создает запрос на покупку товара и чат между покупателем и продавцом.
func (c *Chat) CreateOrderRequest(
	ctx context.Context,
	adID int64,
	buyerID int64,
) error {
	c.log.InfoContext(ctx, "creating order request",
		slog.String("op", opCreateOrderRequest),
		slog.Int64("ad_id", adID),
		slog.Int64("buyer_id", buyerID),
	)

	// Получаем объявление
	ad, err := c.adStorage.GetAdByID(ctx, adID)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to get ad",
			slog.String("op", opCreateOrderRequest),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Проверяем статус
	if ad.Status != models.AdStatusActive {
		c.log.WarnContext(ctx, "ad is not active",
			slog.String("op", opCreateOrderRequest),
			slog.String("status", ad.Status),
		)
		return fmt.Errorf("ad is not active")
	}

	// Проверяем, что покупатель не является продавцом
	if buyerID == ad.SellerID {
		c.log.WarnContext(ctx, "buyer and seller are the same",
			slog.String("op", opCreateOrderRequest),
		)
		return fmt.Errorf("cannot buy own product")
	}

	// Получяем или создаём чат
	chatID, err := c.chatStorage.GetOrCreateChat(ctx, adID, buyerID, ad.SellerID)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to get or create chat",
			slog.String("op", opCreateOrderRequest),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Создём сообщение типа 'order'
	messageText := fmt.Sprintf(
		"Покупатель хочет приобрести товар: %s (Цена: %d)",
		ad.Title,
		ad.Price,
	)

	msg := &models.Message{
		ChatID:   chatID,
		SenderID: buyerID,
		Text:     messageText,
		Type:     models.MessageTypeOrder,
	}

	_, err = c.chatStorage.CreateMessage(ctx, msg)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to create message",
			slog.String("op", opCreateOrderRequest),
			slog.String("error", err.Error()),
		)
		return err
	}

	c.log.InfoContext(ctx, "order request created successfully",
		slog.String("op", opCreateOrderRequest),
		slog.Int64("chat_id", chatID),
	)

	return nil
}

// ConfirmPurchase подтверждает покупку и изменяет статус объявления на 'sold'.
func (c *Chat) ConfirmPurchase(
	ctx context.Context,
	adID int64,
	userID int64,
) error {
	c.log.InfoContext(ctx, "confirming purchase",
		slog.String("op", opConfirmPurchase),
		slog.Int64("ad_id", adID),
		slog.Int64("user_id", userID),
	)

	// Получаем объявление
	ad, err := c.adStorage.GetAdByID(ctx, adID)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to get ad",
			slog.String("op", opConfirmPurchase),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Проверяем, что пользователь - продавец
	if userID != ad.SellerID {
		c.log.WarnContext(ctx, "user is not the seller",
			slog.String("op", opConfirmPurchase),
			slog.Int64("user_id", userID),
			slog.Int64("seller_id", ad.SellerID),
		)
		return fmt.Errorf("forbidden: not the seller")
	}

	// Обновляем статус объявления на 'sold'
	err = c.chatStorage.UpdateAdStatus(ctx, adID, models.AdStatusSold)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to update ad status",
			slog.String("op", opConfirmPurchase),
			slog.String("error", err.Error()),
		)
		return err
	}

	c.log.InfoContext(ctx, "purchase confirmed successfully",
		slog.String("op", opConfirmPurchase),
		slog.Int64("ad_id", adID),
	)

	return nil
}
