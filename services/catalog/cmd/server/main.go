// Catalog service entry point.
//
// HTTP роуты для объявлений + gRPC сервер CatalogService для Commerce.
// Зависимости: PostgreSQL (общая БД), Redis (view-pipeline), S3 (фото), Kafka (события),
// Auth gRPC (валидация JWT в middleware).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/logger"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/metrics"
	catalogv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/catalog/v1"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/config"
	cataloggrpc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/handlers"
	catalogkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/kafka"
	catalogmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/delivery/middleware"
	adrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/ad"
	platformrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/platform_setting"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/s3"
	sysmsgrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/system_message"
	viewrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/view"
	viewcache "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/view_cache"
	viewstream "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/repository/view_stream"
	adsusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/usecase/ads"
	moderationuc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/usecase/moderation"
	viewsusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/catalog/internal/usecase/views"

	grpclib "google.golang.org/grpc"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logger.SetupLogger(cfg.Env)
	log.Info("catalog service starting",
		slog.Int("http_port", cfg.HTTP.Port),
		slog.Int("grpc_port", cfg.GRPC.Port),
	)

	pg, err := postgres.New(cfg.DatabaseDSN)
	if err != nil {
		log.Error("postgres connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pg.Close()

	redisCache := redis.New(cfg.RedisAddr)
	defer func() { _ = redisCache.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s3Client, err := s3.NewS3Client(ctx, cfg.S3Storage)
	if err != nil {
		log.Error("s3 init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	authClient, err := cataloggrpc.NewAuthClient(cfg.AuthGRPCAddr)
	if err != nil {
		log.Error("auth grpc client failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = authClient.Close() }()

	adStorage := adrepo.NewAdStorage(pg.Pool, log)
	viewStorage := viewrepo.NewViewStorage(pg.Pool, log)
	viewCache := viewcache.New(redisCache.Pool(), cfg.Views.DedupTTL, cfg.Views.CountCacheTTL)
	viewStream := viewstream.New(redisCache.Pool(), log, cfg.Views.StreamKey, cfg.Views.ConsumerGroup, "catalog-worker")
	if err := viewStream.EnsureConsumerGroup(); err != nil {
		log.Warn("failed to ensure views consumer group (will retry on consumer start)", slog.String("error", err.Error()))
	}

	brokers := splitBrokers(cfg.Kafka.Brokers)
	kafkaProducer := catalogkafka.NewProducer(brokers, log)
	defer func() { _ = kafkaProducer.Close() }()

	kafkaConsumer := catalogkafka.NewConsumer(brokers, cfg.Kafka.GroupID, log)
	defer func() { _ = kafkaConsumer.Close() }()

	platformStorage := platformrepo.New(pg.Pool, log)
	sysMessenger := sysmsgrepo.New(pg.Pool, log)

	moderationGate := moderationuc.NewGate(platformStorage, log, 30*time.Second)
	bootCtx, bootCancel := context.WithTimeout(ctx, 5*time.Second)
	systemUserID, err := moderationuc.LoadSystemUserID(bootCtx, platformStorage)
	bootCancel()
	if err != nil {
		log.Warn("system user id not configured; admin notifications will be skipped",
			slog.String("error", err.Error()),
		)
	}

	adsUC := adsusecase.
		New(log, adStorage, s3Client, cfg.Search, kafkaProducer).
		WithAdmin(sysMessenger, moderationGate, systemUserID)
	viewsUC := viewsusecase.New(log, viewCache, viewStream, viewStream, viewStorage, viewsusecase.Config{
		BatchSize:     cfg.Views.BatchSize,
		FlushInterval: cfg.Views.FlushInterval,
		BlockTimeout:  cfg.Views.BlockTimeout,
	})

	adsHandlers := handlers.NewAdsHandlers(log, adsUC)
	viewsHandlers := handlers.NewViewsHandlers(log, viewsUC)

	httpSrv := buildHTTPServer(log, cfg.HTTP.Port, adsHandlers, viewsHandlers, authClient)
	grpcSrv := buildGRPCServer(log, adStorage, kafkaProducer)

	httpErr := make(chan error, 1)
	grpcErr := make(chan error, 1)

	go func() {
		log.Info("http server listening", slog.String("addr", httpSrv.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErr <- err
		}
	}()

	go func() {
		addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
		lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
		if err != nil {
			grpcErr <- fmt.Errorf("grpc listen %s: %w", addr, err)
			return
		}
		log.Info("grpc server listening", slog.String("addr", addr))
		if err := grpcSrv.Serve(lis); err != nil {
			grpcErr <- err
		}
	}()

	go kafkaConsumer.Run(ctx)
	go viewsUC.RunConsumer(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-stop:
		log.Info("received signal", slog.String("signal", sig.String()))
	case err := <-httpErr:
		log.Error("http server failed", slog.String("error", err.Error()))
	case err := <-grpcErr:
		log.Error("grpc server failed", slog.String("error", err.Error()))
	}

	log.Info("shutting down")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", slog.String("error", err.Error()))
	}
	grpcSrv.GracefulStop()
	log.Info("catalog service stopped")
}

func buildHTTPServer(
	log *slog.Logger,
	port int,
	ads *handlers.AdsHandlers,
	views *handlers.ViewsHandlers,
	authClient *cataloggrpc.AuthClient,
) *http.Server {
	mux := http.NewServeMux()
	prefix := api.ApiPrefix

	authMW := catalogmw.GRPCAuthMiddleware(log, authClient)
	optionalAuthMW := catalogmw.GRPCOptionalAuthMiddleware(log, authClient)

	// Публичные ручки
	mux.HandleFunc("GET "+prefix+"/ads", ads.HandleGetAds)
	mux.HandleFunc("GET "+prefix+"/ads/search", ads.HandleSearchAds)
	mux.HandleFunc("GET "+prefix+"/ads/{id}", ads.HandleGetAdByID)
	mux.HandleFunc("GET "+prefix+"/ads/{id}/price-history", ads.HandleGetPriceHistory)
	mux.HandleFunc("GET "+prefix+"/categories", ads.HandleGetCategories)
	mux.HandleFunc("GET "+prefix+"/categories/{id}/characteristics", ads.HandleGetCategoryCharacteristics)

	// Просмотры — опциональная авторизация
	mux.Handle("POST "+prefix+"/ads/{id}/view", optionalAuthMW(http.HandlerFunc(views.HandleRecordView)))

	// Защищённые ручки
	mux.Handle("POST "+prefix+"/ads", authMW(http.HandlerFunc(ads.HandleCreateAd)))
	mux.Handle("PUT "+prefix+"/ads/{id}", authMW(http.HandlerFunc(ads.HandleUpdateAdByID)))
	mux.Handle("DELETE "+prefix+"/ads/{id}", authMW(http.HandlerFunc(ads.HandleDeleteAd)))
	mux.Handle("POST "+prefix+"/ads/{id}/close", authMW(http.HandlerFunc(ads.HandleCloseAdByID)))
	mux.Handle("POST "+prefix+"/ads/{id}/favorite", authMW(http.HandlerFunc(ads.HandleAddToFavorites)))
	mux.Handle("DELETE "+prefix+"/ads/{id}/favorite", authMW(http.HandlerFunc(ads.HandleDeleteFromFavorites)))
	mux.Handle("GET "+prefix+"/profile/favorites", authMW(http.HandlerFunc(ads.HandleGetFavorites)))

	// /users/{id}/ads с опциональной авторизацией — нужна для проверки доступа к ?tab=pending
	mux.Handle("GET "+prefix+"/users/{id}/ads", optionalAuthMW(http.HandlerFunc(ads.HandleGetUserAds)))

	// Админ-ручки: auth + require role=admin.
	adminOnly := sharedmw.RequireRole("admin")
	mux.Handle("DELETE "+prefix+"/ads/{id}/admin",
		authMW(adminOnly(http.HandlerFunc(ads.HandleAdminDeleteAd))))
	mux.Handle("GET "+prefix+"/admin/moderation/settings",
		authMW(adminOnly(http.HandlerFunc(ads.HandleGetModerationFlag))))
	mux.Handle("PUT "+prefix+"/admin/moderation/settings",
		authMW(adminOnly(http.HandlerFunc(ads.HandleSetModerationFlag))))
	mux.Handle("GET "+prefix+"/admin/moderation/queue",
		authMW(adminOnly(http.HandlerFunc(ads.HandleGetModerationQueue))))
	mux.Handle("POST "+prefix+"/admin/moderation/ads/{id}/approve",
		authMW(adminOnly(http.HandlerFunc(ads.HandleApproveAd))))
	mux.Handle("POST "+prefix+"/admin/moderation/ads/{id}/reject",
		authMW(adminOnly(http.HandlerFunc(ads.HandleRejectAd))))

	// /metrics - Prometheus scrape endpoint, в обход CSRF и AccessLog (см. middleware/access_log.go).
	mux.Handle("GET /metrics", metrics.Handler())

	httpMetrics := metrics.New("catalog")

	// Цепочка middleware (снаружи внутрь): CORS -> RequestID -> Metrics -> AccessLog -> CSRF -> mux
	handler := sharedmw.CSRFMiddleware(mux)
	handler = sharedmw.AccessLogMiddleware(log)(handler)
	handler = httpMetrics.Middleware(handler)
	handler = sharedmw.RequestIDMiddleware(handler)
	handler = sharedmw.CORSMiddleware(handler)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func buildGRPCServer(log *slog.Logger, ads cataloggrpc.AdProvider, publisher cataloggrpc.EventPublisher) *grpclib.Server {
	grpcMetrics := metrics.NewGRPC("catalog")
	srv := grpclib.NewServer(grpclib.UnaryInterceptor(grpcMetrics.UnaryServerInterceptor()))
	catalogv1.RegisterCatalogServiceServer(srv, cataloggrpc.NewServer(log, ads, publisher))
	return srv
}

func splitBrokers(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}
