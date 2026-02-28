// Пакет logger настраивает глобальный логгер в зависимости от окружения.
// Поддерживаются вывод в текстовом или JSON формате с уровнями логирования.
package logger

import (
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

// envLocal, envDev и envProd обозначают возможные значения переменной
// окружения для выбора конфигурации логгера.
func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		log = slog.Default()
	}
	return log
}

// SetupLogger возвращает *slog.Logger, сконфигурированный под указанное
// окружение. В локальном режиме выводится человекочитаемый текст, в других
// — формируется JSON.
