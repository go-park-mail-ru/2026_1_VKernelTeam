// Package app инициализирует компоненты приложения и связывает
// их между собой. В частности, создаётся хранилище, сервис auth и HTTP
// сервер.
package app

import (
	"log/slog"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/config"
)

// App содержит корневые объекты приложения, например HTTP-сервер.
type App struct {
	HTTPServer *httpapp.App
}

// New собирает все зависимости и возвращает готовое приложение.
func New(
	log *slog.Logger,
	cfg *config.Config,
) *App {
	storage, err := storage.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to initialize storage", "err", err)
		panic(err)
	}

	// инициализируем чёрный список
	tokenBlacklist := blacklist.New(cfg.CleanupInterval)

	// создаём сервис Auth
	authService := auth.New(log, storage, storage, tokenBlacklist, cfg.TokenTTL, cfg.TokenSecret)

	// создеём сервис Ads
	adsService := ads.NewAdsRepository()

	services := httpapp.Services{
		Ads:  adsService,
		Auth: authService,
	}

	// создаём HTTP-приложение
	httpApp := httpapp.New(log, services, tokenBlacklist, cfg.HTTP.Port, cfg.TokenTTL, cfg.TokenSecret)

	return &App{
		HTTPServer: httpApp,
	}
}
