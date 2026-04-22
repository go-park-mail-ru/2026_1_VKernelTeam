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
	GetChatByID(ctx context.Context, chatID int64) (models.Chat, error)
	CompletePurchase(ctx context.Context, buyerID int64, productID int64, price int64) error
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

// CreateOrderRequest создает запрос на покупку товара и чат между покупателем
// и продавцом. Возвращает ID чата (существующего или свежесозданного).
func (c *Chat) CreateOrderRequest(
	ctx context.Context,
	adID int64,
	buyerID int64,
) (int64, error) {
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
		return 0, err
	}

	// Проверяем статус
	if ad.Status != models.AdStatusActive {
		c.log.WarnContext(ctx, "ad is not active",
			slog.String("op", opCreateOrderRequest),
			slog.String("status", ad.Status),
		)
		return 0, fmt.Errorf("ad is not active")
	}

	// Проверяем, что покупатель не является продавцом
	if buyerID == ad.SellerID {
		c.log.WarnContext(ctx, "buyer and seller are the same",
			slog.String("op", opCreateOrderRequest),
		)
		return 0, fmt.Errorf("cannot buy own product")
	}

	// Получяем или создаём чат
	chatID, err := c.chatStorage.GetOrCreateChat(ctx, adID, buyerID, ad.SellerID)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to get or create chat",
			slog.String("op", opCreateOrderRequest),
			slog.String("error", err.Error()),
		)
		return 0, err
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

	// Сохраняем сообщение в чате
	if _, err = c.chatStorage.CreateMessage(ctx, msg); err != nil {
		c.log.ErrorContext(ctx, "failed to create message",
			slog.String("op", opCreateOrderRequest),
			slog.String("error", err.Error()),
		)
		return 0, err
	}

	c.log.InfoContext(ctx, "order request created successfully",
		slog.String("op", opCreateOrderRequest),
		slog.Int64("chat_id", chatID),
	)

	return chatID, nil
}

// ConfirmPurchase подтверждает сделку по чату: создаёт заказ для покупателя
// из чата, переводит объявление в 'sold' и удаляет его из корзин всех пользователей.
func (c *Chat) ConfirmPurchase(
	ctx context.Context,
	chatID int64,
	userID int64,
) error {
	c.log.InfoContext(ctx, "confirming purchase",
		slog.String("op", opConfirmPurchase),
		slog.Int64("chat_id", chatID),
		slog.Int64("user_id", userID),
	)

	// получаем чат
	chat, err := c.chatStorage.GetChatByID(ctx, chatID)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to get chat",
			slog.String("op", opConfirmPurchase),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Проверяем, что пользователь является продавцом в этом чате
	if userID != chat.SellerID {
		c.log.WarnContext(ctx, "user is not the seller of the chat",
			slog.String("op", opConfirmPurchase),
			slog.Int64("user_id", userID),
			slog.Int64("seller_id", chat.SellerID),
		)
		return fmt.Errorf("forbidden: not the seller")
	}

	// Получаем объявление
	ad, err := c.adStorage.GetAdByID(ctx, chat.AdID)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to get ad",
			slog.String("op", opConfirmPurchase),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Проверяем статус
	if ad.Status != models.AdStatusActive {
		c.log.WarnContext(ctx, "ad is not active",
			slog.String("op", opConfirmPurchase),
			slog.String("status", ad.Status),
		)
		return fmt.Errorf("ad is not active")
	}

	// Подтверждаем покупку: создаём заказ, переводим объявление в 'sold' и удаляем из корзин
	if err = c.chatStorage.CompletePurchase(ctx, chat.BuyerID, chat.AdID, ad.Price); err != nil {
		c.log.ErrorContext(ctx, "failed to complete purchase",
			slog.String("op", opConfirmPurchase),
			slog.String("error", err.Error()),
		)
		return err
	}

	c.log.InfoContext(ctx, "purchase confirmed successfully",
		slog.String("op", opConfirmPurchase),
		slog.Int64("chat_id", chatID),
		slog.Int64("ad_id", chat.AdID),
		slog.Int64("buyer_id", chat.BuyerID),
	)

	return nil
}
