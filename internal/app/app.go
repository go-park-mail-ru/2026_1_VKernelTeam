// Package app инициализирует компоненты приложения и связывает
// их между собой. В частности, создаётся хранилище, сервис auth и HTTP
// сервер.
package app

import (
	"context"
	"log/slog"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers"
	blacklist "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/database"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
)

type App struct {
	HTTPServer *httpapp.App
	Blacklist  *blacklist.InMemory
	pgStorage  *db.Storage
}

// New собирает все зависимости и возвращает готовое приложение.
func New(
	ctx context.Context,
	log *slog.Logger,
	cfg *config.Config,
) *App {
	storage, err := db.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Error("failed to initialize storage", "err", err)
		panic(err)
	}

	// инициализируем чёрный список
	tokenBlacklist := blacklist.New(cfg.CleanupInterval)

	// создаём сервис Auth
	authService := auth.New(log, storage, tokenBlacklist, cfg.TokenTTL, cfg.TokenSecret)

	// pgStorage реализует интерфейс handlers.Ads (метод GetAll)
	services := handlers.Services{
		Ads:  storage,
		Auth: authService,
	}

	// создаём HTTP-приложение
	httpApp := httpapp.New(log, services, tokenBlacklist, cfg.HTTP.Port, cfg.TokenTTL, cfg.TokenSecret)

	return &App{
		HTTPServer: httpApp,
		Blacklist:  tokenBlacklist,
		pgStorage:  storage,
	}
}

// Stop корректно завершает работу всех компонентов приложения
func (a *App) Stop() {
	a.Blacklist.Stop()
	a.HTTPServer.Stop()
	a.pgStorage.Close()
}
