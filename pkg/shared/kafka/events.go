// Package kafka — общий клиент для всех сервисов Clover.
//
// Producer/Consumer обёртки над segmentio/kafka-go, единый формат сообщений,
// константы топиков и типов событий, чтобы исключить рассинхрон между сервисами.
package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Топики Kafka. Имена согласованы для всех сервисов.
const (
	TopicAuthUserEvents  = "clover.auth.user-events"
	TopicCatalogAdEvents = "clover.catalog.ad-events"
)

// Типы событий.
const (
	EventUserUpdated = "user.updated"
	EventUserDeleted = "user.deleted"
	EventAdSold      = "ad.sold"
	EventAdDeleted   = "ad.deleted"
)

// Event — общий формат сообщения в Kafka.
// Payload хранится в RawMessage, чтобы каждая сторона декодировала свою схему сама.
type Event struct {
	EventType string          `json:"event_type"`
	EventID   string          `json:"event_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent сериализует payload и собирает Event с уникальным ID и текущим временем.
func NewEvent(eventType string, payload any) (Event, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("kafka.NewEvent: marshal payload: %w", err)
	}
	return Event{
		EventType: eventType,
		EventID:   uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Payload:   raw,
	}, nil
}

// UnmarshalPayload декодирует Payload в указанную структуру.
func (e Event) UnmarshalPayload(out any) error {
	if err := json.Unmarshal(e.Payload, out); err != nil {
		return fmt.Errorf("kafka.Event.UnmarshalPayload: %w", err)
	}
	return nil
}

// UserPayload — payload для событий user.* (updated/deleted).
type UserPayload struct {
	UserID int64 `json:"user_id"`
}

// AdPayload — payload для событий ad.* (sold/deleted).
type AdPayload struct {
	AdID    int64 `json:"ad_id"`
	BuyerID int64 `json:"buyer_id,omitempty"`
}
