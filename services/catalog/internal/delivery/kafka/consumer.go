// Package kafka — domain-обёртка над общим Kafka consumer для catalog.
//
// Подписывается на clover.auth.user-events. Сейчас БД делает hard cleanup
// объявлений удалённого пользователя через ON DELETE CASCADE на product.seller_id,
// поэтому handler user.deleted ничего не делает кроме лога. Когда у catalog
// появится Redis-кэш продавцов — сюда добавится инвалидация.
package kafka

import (
	"context"
	"log/slog"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
)

// Consumer слушает события пользователей из Auth.
type Consumer struct {
	inner *sharedkafka.Consumer
	log   *slog.Logger
}

// NewConsumer создаёт consumer и регистрирует обработчики user.deleted/user.updated.
func NewConsumer(brokers []string, groupID string, log *slog.Logger) *Consumer {
	c := &Consumer{
		inner: sharedkafka.NewConsumer(brokers, sharedkafka.TopicAuthUserEvents, groupID, log),
		log:   log,
	}
	c.inner.On(sharedkafka.EventUserDeleted, c.handleUserDeleted)
	c.inner.On(sharedkafka.EventUserUpdated, c.handleUserUpdated)
	return c
}

// Run запускает чтение сообщений до отмены контекста.
func (c *Consumer) Run(ctx context.Context) {
	c.inner.Run(ctx)
}

// Close закрывает reader.
func (c *Consumer) Close() error {
	return c.inner.Close()
}

func (c *Consumer) handleUserDeleted(ctx context.Context, event sharedkafka.Event) error {
	var payload sharedkafka.UserPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		return err
	}
	c.log.InfoContext(ctx, "user.deleted received (cleanup handled by postgres CASCADE)",
		slog.Int64("user_id", payload.UserID),
		slog.String("event_id", event.EventID),
	)
	return nil
}

func (c *Consumer) handleUserUpdated(ctx context.Context, event sharedkafka.Event) error {
	var payload sharedkafka.UserPayload
	if err := event.UnmarshalPayload(&payload); err != nil {
		return err
	}
	c.log.DebugContext(ctx, "user.updated received (no seller cache yet, no-op)",
		slog.Int64("user_id", payload.UserID),
	)
	return nil
}
