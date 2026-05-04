package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCtxHandler(buf *bytes.Buffer, level slog.Level) *ContextHandler {
	return &ContextHandler{
		inner: slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level}),
	}
}

func TestContextHandler_HandleAddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(newCtxHandler(&buf, slog.LevelDebug))

	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "req-42")
	log.InfoContext(ctx, "hello")

	out := buf.String()
	assert.Contains(t, out, "req-42")
	assert.Contains(t, out, "request_id")
}

func TestContextHandler_HandleNoRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(newCtxHandler(&buf, slog.LevelDebug))

	log.InfoContext(context.Background(), "hello")

	assert.NotContains(t, buf.String(), "request_id")
}

func TestContextHandler_Enabled(t *testing.T) {
	h := newCtxHandler(&bytes.Buffer{}, slog.LevelWarn)

	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

func TestContextHandler_WithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	root := newCtxHandler(&buf, slog.LevelDebug)

	withAttrs := root.WithAttrs([]slog.Attr{slog.String("svc", "auth")})
	require.NotNil(t, withAttrs)
	_, ok := withAttrs.(*ContextHandler)
	assert.True(t, ok, "WithAttrs must keep ContextHandler wrapper")

	withGroup := withAttrs.WithGroup("g")
	require.NotNil(t, withGroup)
	_, ok = withGroup.(*ContextHandler)
	assert.True(t, ok, "WithGroup must keep ContextHandler wrapper")

	slog.New(withAttrs).Info("hello")
	assert.Contains(t, buf.String(), "auth")
}
