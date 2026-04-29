// Package kafka реализует Kafka producer для Auth-сервиса.
//
// Публикует события в топик clover.auth.user-events:
//   - user.updated — профиль обновлён (для инвалидации кэшей)
//   - user.deleted — пользователь удалён (для очистки данных в других сервисах)
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

// Event описывает формат сообщения для Kafka.
type Event struct {
	EventType string      `json:"event_type"`
	EventID   string      `json:"event_id"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// UserPayload — payload для событий пользователя.
type UserPayload struct {
	UserID int64 `json:"user_id"`
}

// Producer публикует события пользователей в Kafka.
type Producer struct {
	writer *kafkago.Writer
	log    *slog.Logger
}

// NewProducer создаёт Kafka producer для указанного топика.
func NewProducer(brokers []string, topic string, log *slog.Logger) *Producer {
	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.Hash{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireOne,
	}

	return &Producer{
		writer: writer,
		log:    log,
	}
}

// Close закрывает Kafka writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// PublishUserUpdated публикует событие user.updated.
func (p *Producer) PublishUserUpdated(ctx context.Context, userID int64) error {
	return p.publish(ctx, "user.updated", userID)
}

// PublishUserDeleted публикует событие user.deleted.
func (p *Producer) PublishUserDeleted(ctx context.Context, userID int64) error {
	return p.publish(ctx, "user.deleted", userID)
}

func (p *Producer) publish(ctx context.Context, eventType string, userID int64) error {
	event := Event{
		EventType: eventType,
		EventID:   uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Payload:   UserPayload{UserID: userID},
	}

	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka.Producer.publish: marshal: %w", err)
	}

	msg := kafkago.Message{
		Key:   []byte(strconv.FormatInt(userID, 10)),
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.ErrorContext(ctx, "failed to publish kafka event",
			slog.String("event_type", eventType),
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("kafka.Producer.publish: write: %w", err)
	}

	p.log.InfoContext(ctx, "kafka event published",
		slog.String("event_type", eventType),
		slog.Int64("user_id", userID),
		slog.String("event_id", event.EventID),
	)

	return nil
}
