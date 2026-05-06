// Package kafka — domain-обёртка над общим Kafka producer.
//
// Реализует EventPublisher из usecase auth: PublishUserUpdated/PublishUserDeleted.
// Низкоуровневая работа с Kafka — в pkg/shared/kafka.
package kafka

import (
	"context"
	"log/slog"
	"strconv"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
)

// Producer публикует события пользователей в clover.auth.user-events.
type Producer struct {
	inner *sharedkafka.Producer
}

// NewProducer создаёт producer, подключённый к топику clover.auth.user-events.
func NewProducer(brokers []string, log *slog.Logger) *Producer {
	return &Producer{
		inner: sharedkafka.NewProducer(brokers, sharedkafka.TopicAuthUserEvents, log),
	}
}

// Close закрывает Kafka writer.
func (p *Producer) Close() error {
	return p.inner.Close()
}

// PublishUserUpdated публикует событие user.updated с ключом user_id.
func (p *Producer) PublishUserUpdated(ctx context.Context, userID int64) error {
	return p.publishUserEvent(ctx, sharedkafka.EventUserUpdated, userID)
}

// PublishUserDeleted публикует событие user.deleted с ключом user_id.
func (p *Producer) PublishUserDeleted(ctx context.Context, userID int64) error {
	return p.publishUserEvent(ctx, sharedkafka.EventUserDeleted, userID)
}

func (p *Producer) publishUserEvent(ctx context.Context, eventType string, userID int64) error {
	key := strconv.FormatInt(userID, 10)
	return p.inner.Publish(ctx, key, eventType, sharedkafka.UserPayload{UserID: userID})
}
