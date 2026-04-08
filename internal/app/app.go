// Package app инициализирует компоненты приложения и связывает
// их между собой. В частности, создаётся хранилище, сервис auth и HTTP
// сервер.
package app

import (
	"context"
	"log/slog"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/ad"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/cart"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/refresh"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	cartUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/cart"
)

type App struct {
	HTTPServer *httpapp.App
	RedisCache *redis.RedisCache
	dbClient   *postgres.Client
}

// New собирает все зависимости и возвращает готовое приложение.
func New(
	log *slog.Logger,
	cfg *config.Config,
	baseDb ...*postgres.Client,
) *App {
	var dbClient *postgres.Client
	var err error

	if len(baseDb) > 0 && baseDb[0] != nil {
		dbClient = baseDb[0] // Используем заглушку из теста
	} else {
		dbClient, err = postgres.New(cfg.DatabaseDSN)
		if err != nil {
			log.Error("failed to initialize postgres client", "err", err)
			panic(err)
		}
	}

	userRepo := user.NewUserStorage(dbClient.Pool)
	adRepo := ad.NewAdStorage(dbClient.Pool)

	// инициализируем redis
	rc := redis.New(cfg.RedisAddr)
	bl := blacklist.New(rc)
	ref := refresh.New(rc)

	// инициализируем S3
	s3Storage, err := s3.NewS3Client(context.Background(), cfg.S3Storage)
	if err != nil {
		log.Error("failed to initialize S3 client", "err", err)
		panic(err)
	}

	// создаём сервис Auth
	authService := auth.New(log, userRepo, bl, ref, s3Storage, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	// создаём сервис Ads
	adsService := ads.New(log, adRepo)

	// создаём сервис корзины
	cartRepo := cart.NewCartStorage(dbClient.Pool)
	cartService := cartUC.New(log, cartRepo, adRepo)

	services := httpapp.Services{
		Ads:  adsService,
		Auth: authService,
		Cart: cartService,
	}

	// создаём HTTP-приложение
	httpApp := httpapp.New(log, services, bl, cfg.HTTP.Port, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	return &App{
		HTTPServer: httpApp,
		RedisCache: rc,
		dbClient:   dbClient,
	}
}

// Stop останавливает приложение.
func (a *App) Stop() {
	a.RedisCache.Close()
	a.HTTPServer.Stop()

	if a.dbClient != nil && a.dbClient.Pool != nil {
		a.dbClient.Close()
	}
}
