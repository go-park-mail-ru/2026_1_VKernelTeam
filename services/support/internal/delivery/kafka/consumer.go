// Package kafka — domain-обёртка над общим Kafka consumer.
//
// Подписывается на топик clover.auth.user-events и обрабатывает события user.deleted:
// анонимизирует тикеты и сообщения удалённого пользователя (user_id = 0).
package kafka

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
)

// PgxPool интерфейс для операций с БД (минимум, нужный для анонимизации).
type PgxPool interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

// Consumer слушает события пользователей из Auth-сервиса.
type Consumer struct {
	inner *sharedkafka.Consumer
	pool  PgxPool
	log   *slog.Logger
}

// NewConsumer создаёт consumer и регистрирует обработчик user.deleted.
func NewConsumer(brokers []string, groupID string, pool PgxPool, log *slog.Logger) *Consumer {
	c := &Consumer{
		inner: sharedkafka.NewConsumer(brokers, sharedkafka.TopicAuthUserEvents, groupID, log),
		pool:  pool,
		log:   log,
	}
	c.inner.On(sharedkafka.EventUserDeleted, c.handleUserDeleted)
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

	c.log.InfoContext(ctx, "processing user.deleted",
		slog.Int64("user_id", payload.UserID),
		slog.String("event_id", event.EventID),
	)

	const ticketQuery = `UPDATE support_ticket SET user_id = 0 WHERE user_id = $1`
	if _, err := c.pool.Exec(ctx, ticketQuery, payload.UserID); err != nil {
		return fmt.Errorf("anonymize tickets: %w", err)
	}

	const messageQuery = `UPDATE support_message SET user_id = 0 WHERE user_id = $1`
	if _, err := c.pool.Exec(ctx, messageQuery, payload.UserID); err != nil {
		return fmt.Errorf("anonymize messages: %w", err)
	}

	c.log.InfoContext(ctx, "user data anonymized", slog.Int64("user_id", payload.UserID))
	return nil
}
