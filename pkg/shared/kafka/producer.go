package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mailru/easyjson"
	kafkago "github.com/segmentio/kafka-go"
)

// Producer публикует события в один топик.
type Producer struct {
	writer *kafkago.Writer
	topic  string
	log    *slog.Logger
}

// NewProducer создаёт producer для указанного топика.
func NewProducer(brokers []string, topic string, log *slog.Logger) *Producer {
	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.Hash{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireOne,
	}
	return &Producer{writer: writer, topic: topic, log: log}
}

// Close закрывает writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Publish сериализует payload в Event и пишет его в Kafka с ключом key
// (используется для партицирования: один и тот же ключ всегда уходит в одну партицию).
func (p *Producer) Publish(ctx context.Context, key, eventType string, payload any) error {
	event, err := NewEvent(eventType, payload)
	if err != nil {
		return err
	}

	value, err := easyjson.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka.Producer.Publish: marshal event: %w", err)
	}

	msg := kafkago.Message{
		Key:   []byte(key),
		Value: value,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.ErrorContext(ctx, "kafka publish failed",
			slog.String("topic", p.topic),
			slog.String("event_type", eventType),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("kafka.Producer.Publish: write: %w", err)
	}

	p.log.InfoContext(ctx, "kafka event published",
		slog.String("topic", p.topic),
		slog.String("event_type", eventType),
		slog.String("event_id", event.EventID),
		slog.String("key", key),
	)
	return nil
}
