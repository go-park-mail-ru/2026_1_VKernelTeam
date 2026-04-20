package chat

//go:generate mockgen -source=chat.go -destination=mocks/mock_chat.go -package=mocks

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/dto"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/domain/models"
)

// const (
// 	opCreateOrderRequest = "usecase.chat.CreateOrderRequest"
// 	opConfirmPurchase    = "usecase.chat.ConfirmPurchase"
// )

// ChatProvider описывает интерфейс для работы с чатами в репозитории
type ChatProvider interface {
	GetOrCreateChat(ctx context.Context, adID int64, buyerID int64, sellerID int64) (int64, error)
	CreateMessage(ctx context.Context, message *models.Message) (int64, error)
	UpdateAdStatus(ctx context.Context, adID int64, status string) error
}

// AdProvider описывает интерфейс для получения данных объявления
type AdProvider interface {
	GetAdByID(ctx context.Context, id int64) (models.Ad, error)
	UpdateAd(ctx context.Context, req *dto.UpdateAdRequest) error
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

// CreateOrderRequest создает запрос на покупку товара (отправляет системное уведомление продавцу).
// Этап 1: Получает объявление по ID
// Этап 2: Проверяет, что объявление в статусе 'active'
// Этап 3: Убедится, что покупатель не равен продавцу
// Этап 4: Получает/создает чат между покупателем и продавцом
// Этап 5: Создает сообщение типа 'order' с текстом "Пользователь хочет купить ваш товар"
// Этап 6: Опционально переводит объявление в статус 'reserved'
func (c *Chat) CreateOrderRequest(
	ctx context.Context,
	adID int64,
	buyerID int64,
) error {
	// TODO: Получить объявление по ID
	// TODO: Проверить, что объявление в статусе 'active'
	// TODO: Проверить, что buyerID != sellerID
	// TODO: Получить/создать чат
	// TODO: Создать сообщение типа 'order'
	return nil
}

// ConfirmPurchase подтверждает покупку и изменяет статус объявления на 'sold'.
// Этап 1: Получает объявление по ID
// Этап 2: Проверяет, что userID является продавцом (sellerID)
// Этап 3: Обновляет статус объявления на 'sold'
func (c *Chat) ConfirmPurchase(
	ctx context.Context,
	adID int64,
	userID int64,
) error {
	// TODO: Получить объявление по ID
	// TODO: Проверить, что userID == ad.SellerID (или вернуть ошибку Forbidden)
	// TODO: Обновить статус объявления на 'sold'
	return nil
}
