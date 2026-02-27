// Пакет app инициализирует компоненты приложения и связывает
// их между собой. В частности, создаётся хранилище, сервис auth и HTTP
// сервер.
package app

import (
	"log/slog"
	"time"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
)

// App содержит корневые объекты приложения, например HTTP-сервер.
type App struct {
	HTTPServer *httpapp.App
}

// New собирает все зависимости и возвращает готовое приложение.
func New(
	log *slog.Logger,
	httpPort int,
	storagePath string,
	tokenTTL time.Duration,
) *App {
	storage, err := storage.New(storagePath)
	if err != nil {
		log.Error("failed to initialize storage", "err", err)
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, tokenTTL)

	httpApp := httpapp.New(log, authService, httpPort)

	return &App{
		HTTPServer: httpApp,
	}
}
