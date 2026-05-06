package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	sharedkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/kafka"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/kafka/mocks"
	"github.com/golang/mock/gomock"
)

func newTestLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newConsumer(t *testing.T, cleaner CartCleaner) *Consumer {
	t.Helper()
	c := NewConsumer([]string{"localhost:9092"}, "test-group", cleaner, newTestLog())
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestNewConsumer(t *testing.T) {
	ctrl := gomock.NewController(t)
	c := newConsumer(t, mocks.NewMockCartCleaner(ctrl))

	if c == nil {
		t.Fatal("expected consumer instance")
	}
}

func TestHandleAdDeleted_CallsRemove(t *testing.T) {
	ctrl := gomock.NewController(t)
	cleaner := mocks.NewMockCartCleaner(ctrl)
	cleaner.EXPECT().RemoveProductFromAllCarts(gomock.Any(), int64(7)).Return(int64(3), nil)

	c := newConsumer(t, cleaner)
	payload, _ := json.Marshal(sharedkafka.AdPayload{AdID: 7})
	err := c.handleAdDeleted(context.Background(), sharedkafka.Event{
		EventType: sharedkafka.EventAdDeleted, EventID: "id-1", Payload: payload,
	})

	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestHandleAdDeleted_PropagatesRemoveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	cleaner := mocks.NewMockCartCleaner(ctrl)
	cleaner.EXPECT().RemoveProductFromAllCarts(gomock.Any(), int64(7)).Return(int64(0), errors.New("db down"))

	c := newConsumer(t, cleaner)
	payload, _ := json.Marshal(sharedkafka.AdPayload{AdID: 7})
	err := c.handleAdDeleted(context.Background(), sharedkafka.Event{
		EventType: sharedkafka.EventAdDeleted, Payload: payload,
	})

	if err == nil {
		t.Fatal("expected error to bubble up")
	}
}

func TestHandleAdDeleted_BadPayloadReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	c := newConsumer(t, mocks.NewMockCartCleaner(ctrl))
	err := c.handleAdDeleted(context.Background(), sharedkafka.Event{Payload: []byte("not-json")})

	if err == nil {
		t.Fatal("expected error on malformed payload")
	}
}
