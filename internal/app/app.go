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
	chatRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/chat"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/refresh"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/s3"
	supportMessageRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/support_message"
	supportTicketRepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/support_ticket"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/user"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/view"
	viewcache "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/view_cache"
	viewstream "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/view_stream"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/ads"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/auth"
	cartUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/cart"
	chatUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/chat"
	supportMessageUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/support_message"
	supportTicketUC "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/support_ticket"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/usecase/views"
)

type App struct {
	HTTPServer    *httpapp.App
	RedisCache    *redis.RedisCache
	dbClient      *postgres.Client
	s3Client      s3.Storage
	viewsCancelFn context.CancelFunc
	viewsDone     <-chan struct{}
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

	userRepo := user.NewUserStorage(dbClient.Pool, log)
	adRepo := ad.NewAdStorage(dbClient.Pool, log)

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
	adsService := ads.New(log, adRepo, s3Storage, cfg.Search)

	// создаём сервис корзины
	cartRepo := cart.NewCartStorage(dbClient.Pool, log)
	cartService := cartUC.New(log, cartRepo, adRepo)

	// создаём сервис чатов и заказов
	chatStorage := chatRepo.NewChatStorage(dbClient.Pool, log)
	chatService := chatUC.New(log, chatStorage, adRepo)

	// создаём сервис техподдержки
	ticketStorage := supportTicketRepo.NewSupportTicketStorage(dbClient.Pool, log)
	ticketService := supportTicketUC.New(log, ticketStorage)

	// создаём сервис сообщений в чате обращения
	messageStorage := supportMessageRepo.NewSupportMessageStorage(dbClient.Pool, log)
	messageService := supportMessageUC.New(log, messageStorage, ticketStorage, userRepo)

	// создаём компоненты просмотров
	viewStorage := view.NewViewStorage(dbClient.Pool, log)
	viewCache := viewcache.New(rc.Pool(), cfg.Views.DedupTTL, cfg.Views.CountCacheTTL)
	viewStream := viewstream.New(rc.Pool(), log, cfg.Views.StreamKey, cfg.Views.ConsumerGroup, "worker-1")

	if err := viewStream.EnsureConsumerGroup(); err != nil {
		log.Warn("failed to ensure views consumer group (will retry on consumer start)", "err", err)
	}

	viewsService := views.New(log, viewCache, viewStream, viewStream, viewStorage, views.Config{
		BatchSize:     cfg.Views.BatchSize,
		FlushInterval: cfg.Views.FlushInterval,
		BlockTimeout:  cfg.Views.BlockTimeout,
	})

	// запускаем consumer в фоне
	viewsCtx, viewsCancel := context.WithCancel(context.Background())
	viewsDone := make(chan struct{})
	go func() {
		defer close(viewsDone)
		viewsService.RunConsumer(viewsCtx)
	}()

	services := httpapp.Services{
		Ads:            adsService,
		Auth:           authService,
		Cart:           cartService,
		Chat:           chatService,
		SupportTicket:  ticketService,
		SupportMessage: messageService,
		Views:          viewsService,
	}

	// создаём HTTP-приложение
	httpApp := httpapp.New(log, services, bl, userRepo, cfg.HTTP.Port, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	return &App{
		HTTPServer:    httpApp,
		RedisCache:    rc,
		dbClient:      dbClient,
		s3Client:      s3Storage,
		viewsCancelFn: viewsCancel,
		viewsDone:     viewsDone,
	}
}

// Stop останавливает приложение.
func (a *App) Stop() {
	// Останавливаем views consumer и ждём завершения финального flush
	if a.viewsCancelFn != nil {
		a.viewsCancelFn()
	}
	if a.viewsDone != nil {
		<-a.viewsDone
	}

	a.RedisCache.Close()
	a.HTTPServer.Stop()

	if a.dbClient != nil && a.dbClient.Pool != nil {
		a.dbClient.Close()
	}
}
