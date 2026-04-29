// Package kafka реализует Kafka consumer для Support-сервиса.
//
// Подписывается на топик clover.auth.user-events и обрабатывает:
//   - user.deleted — анонимизация тикетов и сообщений удалённого пользователя
package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	kafkago "github.com/segmentio/kafka-go"
)

// PgxPool интерфейс для операций с БД.
type PgxPool interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

// Event описывает формат сообщения из Kafka.
type Event struct {
	EventType string          `json:"event_type"`
	EventID   string          `json:"event_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// UserDeletedPayload — payload события user.deleted.
type UserDeletedPayload struct {
	UserID int64 `json:"user_id"`
}

// Consumer слушает события пользователей из Auth-сервиса.
type Consumer struct {
	reader *kafkago.Reader
	pool   PgxPool
	log    *slog.Logger
}

// NewConsumer создаёт consumer для указанного топика и consumer group.
func NewConsumer(brokers []string, topic, groupID string, pool PgxPool, log *slog.Logger) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	return &Consumer{
		reader: reader,
		pool:   pool,
		log:    log,
	}
}

// Run запускает бесконечный цикл чтения сообщений.
// Завершается при отмене контекста.
func (c *Consumer) Run(ctx context.Context) {
	c.log.Info("kafka consumer started",
		slog.String("topic", c.reader.Config().Topic),
		slog.String("group_id", c.reader.Config().GroupID),
	)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("kafka consumer stopped")
				return
			}
			c.log.ErrorContext(ctx, "failed to read kafka message", slog.String("error", err.Error()))
			continue
		}

		c.handleMessage(ctx, msg)
	}
}

// Close закрывает Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) handleMessage(ctx context.Context, msg kafkago.Message) {
	var event Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.log.ErrorContext(ctx, "failed to unmarshal kafka event",
			slog.String("error", err.Error()),
			slog.String("value", string(msg.Value)),
		)
		return
	}

	switch event.EventType {
	case "user.deleted":
		c.handleUserDeleted(ctx, event)
	default:
		c.log.DebugContext(ctx, "ignoring unknown event type",
			slog.String("event_type", event.EventType),
		)
	}
}

// handleUserDeleted анонимизирует данные удалённого пользователя в тикетах и сообщениях.
// Устанавливает user_id = 0 (анонимный пользователь).
func (c *Consumer) handleUserDeleted(ctx context.Context, event Event) {
	var payload UserDeletedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		c.log.ErrorContext(ctx, "failed to unmarshal user.deleted payload",
			slog.String("error", err.Error()),
		)
		return
	}

	c.log.InfoContext(ctx, "processing user.deleted event",
		slog.Int64("user_id", payload.UserID),
		slog.String("event_id", event.EventID),
	)

	// Анонимизация тикетов
	const ticketQuery = `UPDATE support_ticket SET user_id = 0 WHERE user_id = $1`
	if _, err := c.pool.Exec(ctx, ticketQuery, payload.UserID); err != nil {
		c.log.ErrorContext(ctx, "failed to anonymize tickets",
			slog.Int64("user_id", payload.UserID),
			slog.String("error", err.Error()),
		)
		return
	}

	// Анонимизация сообщений
	const messageQuery = `UPDATE support_message SET user_id = 0 WHERE user_id = $1`
	if _, err := c.pool.Exec(ctx, messageQuery, payload.UserID); err != nil {
		c.log.ErrorContext(ctx, "failed to anonymize messages",
			slog.Int64("user_id", payload.UserID),
			slog.String("error", err.Error()),
		)
		return
	}

	c.log.InfoContext(ctx, "user data anonymized successfully",
		slog.Int64("user_id", payload.UserID),
	)
}
