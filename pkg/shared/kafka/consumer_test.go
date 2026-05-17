package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConsumer_AndOn(t *testing.T) {
	c := NewConsumer([]string{kafkaTestBroker}, "topic", "group", discardLogger())
	require.NotNil(t, c)
	require.NotNil(t, c.reader)
	require.NotNil(t, c.handlers)

	called := false
	c.On("evt", func(ctx context.Context, e Event) error {
		called = true
		return nil
	})
	require.Contains(t, c.handlers, "evt")

	// дёрнем зарегистрированный handler через dispatch
	event := Event{EventType: "evt"}
	value, _ := json.Marshal(event)
	c.dispatch(context.Background(), kafkago.Message{Value: value})
	assert.True(t, called)
}

func TestConsumer_Dispatch_BadJSON(t *testing.T) {
	c := NewConsumer([]string{kafkaTestBroker}, "topic", "group", discardLogger())
	// просто не должен паниковать
	c.dispatch(context.Background(), kafkago.Message{Value: []byte("not json")})
}

func TestConsumer_Dispatch_NoHandler(t *testing.T) {
	c := NewConsumer([]string{kafkaTestBroker}, "topic", "group", discardLogger())

	event := Event{EventType: "unknown"}
	value, _ := json.Marshal(event)
	// без зарегистрированного handler — просто DEBUG лог, без падения.
	c.dispatch(context.Background(), kafkago.Message{Value: value})
}

func TestConsumer_Dispatch_HandlerError(t *testing.T) {
	c := NewConsumer([]string{kafkaTestBroker}, "topic", "group", discardLogger())
	c.On("evt", func(ctx context.Context, e Event) error {
		return errors.New("boom")
	})

	event := Event{EventType: "evt"}
	value, _ := json.Marshal(event)
	// handler возвращает ошибку — она логируется, dispatch не паникует.
	c.dispatch(context.Background(), kafkago.Message{Value: value})
}

func TestConsumer_Run_StopsOnContextCancel(t *testing.T) {
	c := NewConsumer([]string{"127.0.0.1:1"}, "topic", "group", discardLogger())
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем — Run должен выйти быстро.

	done := make(chan struct{})
	go func() {
		c.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-context.Background().Done():
	}
}
