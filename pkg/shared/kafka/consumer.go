package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
)

// HandlerFunc обрабатывает одно событие. Если возвращает ошибку — она логируется,
// но сообщение всё равно считается обработанным (offset commit'ится).
type HandlerFunc func(ctx context.Context, event Event) error

// Consumer слушает топик и диспетчеризует события по eventType зарегистрированным
// обработчикам. События неизвестных типов игнорируются с DEBUG-логом.
type Consumer struct {
	reader   *kafkago.Reader
	handlers map[string]HandlerFunc
	log      *slog.Logger
}

// NewConsumer создаёт consumer для указанного топика и consumer group.
func NewConsumer(brokers []string, topic, groupID string, log *slog.Logger) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &Consumer{
		reader:   reader,
		handlers: make(map[string]HandlerFunc),
		log:      log,
	}
}

// On регистрирует обработчик для конкретного eventType. Повторная регистрация перезапишет.
func (c *Consumer) On(eventType string, handler HandlerFunc) {
	c.handlers[eventType] = handler
}

// Run читает сообщения до отмены контекста.
func (c *Consumer) Run(ctx context.Context) {
	cfg := c.reader.Config()
	c.log.Info("kafka consumer started",
		slog.String("topic", cfg.Topic),
		slog.String("group_id", cfg.GroupID),
	)

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("kafka consumer stopped", slog.String("topic", cfg.Topic))
				return
			}
			c.log.ErrorContext(ctx, "kafka read failed", slog.String("error", err.Error()))
			continue
		}
		c.dispatch(ctx, msg)
	}
}

// Close закрывает reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) dispatch(ctx context.Context, msg kafkago.Message) {
	var event Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.log.ErrorContext(ctx, "kafka unmarshal failed",
			slog.String("error", err.Error()),
			slog.String("value", string(msg.Value)),
		)
		return
	}

	handler, ok := c.handlers[event.EventType]
	if !ok {
		c.log.DebugContext(ctx, "no handler for event",
			slog.String("event_type", event.EventType),
			slog.String("event_id", event.EventID),
		)
		return
	}

	if err := handler(ctx, event); err != nil {
		c.log.ErrorContext(ctx, "kafka handler failed",
			slog.String("event_type", event.EventType),
			slog.String("event_id", event.EventID),
			slog.String("error", err.Error()),
		)
	}
}
