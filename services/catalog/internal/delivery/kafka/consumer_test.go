package kafka

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
)

func TestNewConsumer_RegistersHandlers(t *testing.T) {
	c := NewConsumer([]string{"localhost:9092"}, "test-group", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if c == nil {
		t.Fatal("expected consumer")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestHandleUserDeleted_NoOp(t *testing.T) {
	c := NewConsumer([]string{"localhost:9092"}, "test-group", slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer c.Close()

	payload, _ := json.Marshal(sharedkafka.UserPayload{UserID: 7})
	ev := sharedkafka.Event{EventType: sharedkafka.EventUserDeleted, EventID: "id", Payload: payload}
	if err := c.handleUserDeleted(context.Background(), ev); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}

func TestHandleUserUpdated_NoOp(t *testing.T) {
	c := NewConsumer([]string{"localhost:9092"}, "test-group", slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer c.Close()

	payload, _ := json.Marshal(sharedkafka.UserPayload{UserID: 7})
	ev := sharedkafka.Event{EventType: sharedkafka.EventUserUpdated, EventID: "id", Payload: payload}
	if err := c.handleUserUpdated(context.Background(), ev); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
}
