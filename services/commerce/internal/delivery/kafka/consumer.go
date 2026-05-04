// Package kafka - Kafka consumer commerce-сервиса.
//
// Подписан на clover.catalog.ad-events:
//   - ad.deleted - вычищает товар из корзин всех пользователей
package kafka

//go:generate mockgen -source=consumer.go -destination=mocks/mock_consumer.go -package=mocks

import (
	"context"
	"log/slog"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
)

// CartCleaner - интерфейс репозитория корзины, нужный для cleanup'а
// (реализуется cart.CartStorage.RemoveProductFromAllCarts).
type CartCleaner interface {
	RemoveProductFromAllCarts(ctx context.Context, productID int64) (int64, error)
}

// Consumer слушает clover.catalog.ad-events.
type Consumer struct {
	inner *sharedkafka.Consumer
	cart  CartCleaner
	log   *slog.Logger
}

// NewConsumer создаёт consumer и регистрирует обработчик ad.deleted.
func NewConsumer(brokers []string, groupID string, cart CartCleaner, log *slog.Logger) *Consumer {
	c := &Consumer{
		inner: sharedkafka.NewConsumer(brokers, sharedkafka.TopicCatalogAdEvents, groupID, log),
		cart:  cart,
		log:   log,
	}
	c.inner.On(sharedkafka.EventAdDeleted, c.handleAdDeleted)
	return c
}

// Run запускает чтение сообщений до отмены контекста.
func (c *Consumer) Run(ctx context.Context) { c.inner.Run(ctx) }

// Close закрывает reader.
func (c *Consumer) Close() error { return c.inner.Close() }

func (c *Consumer) handleAdDeleted(ctx context.Context, event sharedkafka.Event) error {
	var payload sharedkafka.AdPayload

	if err := event.UnmarshalPayload(&payload); err != nil {
		return err
	}

	removed, err := c.cart.RemoveProductFromAllCarts(ctx, payload.AdID)
	if err != nil {
		return err
	}

	c.log.InfoContext(ctx, "ad.deleted: cart cleanup",
		slog.Int64("ad_id", payload.AdID),
		slog.Int64("removed_rows", removed),
		slog.String("event_id", event.EventID),
	)

	return nil
}
