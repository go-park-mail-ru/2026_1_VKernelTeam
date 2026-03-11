// Package app инициализирует компоненты приложения и связывает
// их между собой. В частности, создаётся хранилище, сервис auth и HTTP
// сервер.
package app

import (
	"log/slog"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/storage/blacklist"
)

type App struct {
	HTTPServer *httpapp.App
	Blacklist  *blacklist.InMemory
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
	authService := auth.New(log, storage, tokenBlacklist, cfg.TokenTTL, cfg.TokenSecret)

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
		Blacklist:  tokenBlacklist,
	}
}

// Stop корректно завершает работу всех компонентов приложения
func (a *App) Stop() {
	a.Blacklist.Stop()
	a.HTTPServer.Stop()
}
