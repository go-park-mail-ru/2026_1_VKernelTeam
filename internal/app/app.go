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
	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/database"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/refresh"
	redisCache "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/cache/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
)

type App struct {
	HTTPServer   *httpapp.App
	RedisCache   *redisCache.RedisCache
	pgStorage    *db.Storage
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

	// инициализируем redis
	rc := redisCache.New(cfg.RedisAddr)
	bl := blacklist.New(rc)
	ref := refresh.New(rc)

	// создаём сервис Auth
	authService := auth.New(log, storage, bl, ref, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	// pgStorage реализует интерфейс handlers.Ads (метод GetAll)
	services := handlers.Services{
		Ads:  storage,
		Auth: authService,
	}

	// создаём HTTP-приложение
	httpApp := httpapp.New(log, services, bl, cfg.HTTP.Port, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	return &App{
		HTTPServer:   httpApp,
		RedisCache:   rc,
		pgStorage:    storage,
	}
}

func (a *App) Stop() {
	a.RedisCache.Close()
	a.HTTPServer.Stop()
	a.pgStorage.Close()
}
