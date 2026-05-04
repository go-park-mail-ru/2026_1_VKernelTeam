package kafka

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewProducer_AndClose(t *testing.T) {
	p := NewProducer([]string{"localhost:9092"}, "test.topic", discardLogger())
	require.NotNil(t, p)
	require.NotNil(t, p.writer)
	assert.Equal(t, "test.topic", p.topic)
	assert.NoError(t, p.Close())
}

func TestProducer_Publish_MarshalError(t *testing.T) {
	p := NewProducer([]string{"localhost:9092"}, "test.topic", discardLogger())
	defer p.Close()

	// функция в payload не сериализуется в JSON.
	err := p.Publish(context.Background(), "key", EventUserUpdated, func() {})
	assert.Error(t, err)
}

func TestProducer_Publish_WriteFails(t *testing.T) {
	// Адрес заведомо закрыт; короткий дедлайн не даст висеть.
	p := NewProducer([]string{"127.0.0.1:1"}, "test.topic", discardLogger())
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := p.Publish(ctx, "k", EventUserUpdated, UserPayload{UserID: 1})
	assert.Error(t, err)
}
