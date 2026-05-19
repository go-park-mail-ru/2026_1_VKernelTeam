// Package kafka — общий клиент для всех сервисов Clover.
//
// Producer/Consumer обёртки над segmentio/kafka-go, единый формат сообщений,
// константы топиков и типов событий, чтобы исключить рассинхрон между сервисами.
package kafka

//go:generate easyjson -all $GOFILE

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mailru/easyjson"
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
// Если payload реализует easyjson.Marshaler — используется быстрая сериализация,
// иначе fallback на encoding/json (для динамических payload без сгенерированных методов).
func NewEvent(eventType string, payload any) (Event, error) {
	var (
		raw []byte
		err error
	)
	if m, ok := payload.(easyjson.Marshaler); ok {
		raw, err = easyjson.Marshal(m)
	} else {
		raw, err = json.Marshal(payload)
	}
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
// Если out реализует easyjson.Unmarshaler — используется быстрая десериализация,
// иначе fallback на encoding/json.
func (e Event) UnmarshalPayload(out any) error {
	var err error
	if u, ok := out.(easyjson.Unmarshaler); ok {
		err = easyjson.Unmarshal(e.Payload, u)
	} else {
		err = json.Unmarshal(e.Payload, out)
	}
	if err != nil {
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
