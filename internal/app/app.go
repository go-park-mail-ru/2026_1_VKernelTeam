// Package app инициализирует компоненты приложения и связывает
// их между собой. В частности, создаётся хранилище, сервис auth и HTTP
// сервер.
package app

import (
	"log/slog"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http"
	redisCache "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/cache/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/delivery/handlers"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	db "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/database"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/refresh"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
)

type App struct {
	HTTPServer *httpapp.App
	RedisCache *redisCache.RedisCache
	dbClient   *db.Client
}

// New собирает все зависимости и возвращает готовое приложение.
func New(
	log *slog.Logger,
	cfg *config.Config,
) *App {
	dbClient, err := db.New(cfg.DatabaseDSN)
	if err != nil {
		log.Error("failed to initialize postgres client", "err", err)
		panic(err)
	}

	userRepo := db.NewUserStorage(dbClient.Pool)
	adRepo := db.NewAdStorage(dbClient.Pool)

	// инициализируем redis
	rc := redisCache.New(cfg.RedisAddr)
	bl := blacklist.New(rc)
	ref := refresh.New(rc)

	// создаём сервис Auth
	authService := auth.New(log, userRepo, bl, ref, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	// создеём сервис Ads
	adsService := ads.New(log, adRepo)

	services := handlers.Services{
		Ads:  adsService,
		Auth: authService,
	}

	// создаём HTTP-приложение
	httpApp := httpapp.New(log, services, bl, cfg.HTTP.Port, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	return &App{
		HTTPServer: httpApp,
		RedisCache: rc,
		dbClient:   dbClient,
	}
}

func (a *App) Stop() {
	a.RedisCache.Close()
	a.HTTPServer.Stop()
	a.dbClient.Close()
}
