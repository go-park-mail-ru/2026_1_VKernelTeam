// Пакет logger настраивает глобальный логгер в зависимости от окружения.
// Поддерживаются вывод в текстовом или JSON формате с уровнями логирования.
// ContextHandler автоматически извлекает request_id из контекста запроса.
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
)

// envLocal, envDev и envProd обозначают возможные значения переменной
// окружения для выбора конфигурации логгера.
const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

// ContextHandler оборачивает slog.Handler и автоматически
// извлекает request_id из контекста, добавляя его в каждую запись лога.
type ContextHandler struct {
	inner slog.Handler
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if requestID, ok := ctx.Value(middleware.RequestIDKey).(string); ok {
		r.AddAttrs(slog.String("request_id", requestID))
	}
	return h.inner.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}

// SetupLogger возвращает *slog.Logger, сконфигурированный под указанное
// окружение. В локальном режиме выводится человекочитаемый текст, в других
// — формируется JSON. Все режимы оборачиваются ContextHandler для
// автоматического извлечения request_id.
func SetupLogger(env string) *slog.Logger {
	var baseHandler slog.Handler

	switch env {
	case envLocal:
		baseHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	case envDev:
		baseHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	case envProd:
		baseHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	default:
		baseHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	return slog.New(&ContextHandler{inner: baseHandler})
}
