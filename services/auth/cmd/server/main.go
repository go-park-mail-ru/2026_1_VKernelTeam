// Auth service entry point.
//
// Поднимает HTTP (REST) и gRPC сервера для сервиса аутентификации:
//   - HTTP: /api/v1/auth/{register,login,logout,refresh}, /api/v1/profile, /api/v1/users/{id}
//   - gRPC: AuthService.{ValidateToken, GetUser, GetUsersByIDs, CheckRole}
//
// Зависимости: PostgreSQL (общая БД), Redis (blacklist/refresh), Kafka (publish user events), S3 (avatars).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/logger"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/metrics"
	authv1 "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/proto/gen/auth/v1"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/config"
	authgrpc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/delivery/handlers"
	authkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/delivery/kafka"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/blacklist"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/redis"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/refresh"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/s3"
	userrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/repository/user"
	authusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/auth/internal/usecase/auth"

	grpclib "google.golang.org/grpc"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logger.SetupLogger(cfg.Env)
	log.Info("auth service starting",
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

	userStorage := userrepo.NewUserStorage(pg.Pool, log)
	blacklistRepo := blacklist.New(redisCache)
	refreshRepo := refresh.New(redisCache)

	brokers := splitBrokers(cfg.Kafka.Brokers)
	kafkaProducer := authkafka.NewProducer(brokers, log)
	defer func() { _ = kafkaProducer.Close() }()

	authUC := authusecase.New(
		log,
		userStorage,
		blacklistRepo,
		refreshRepo,
		s3Client,
		kafkaProducer,
		cfg.TokenTTL,
		cfg.RefreshTTL,
		cfg.TokenSecret,
	)

	authHandlers := handlers.NewAuthHandlers(log, authUC, cfg.TokenTTL, cfg.RefreshTTL, cfg.TokenSecret)

	httpSrv := buildHTTPServer(log, cfg.HTTP.Port, authHandlers, blacklistRepo, cfg.TokenSecret)
	grpcSrv := buildGRPCServer(log, userStorage, authUC)

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
		lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
		if err != nil {
			grpcErr <- fmt.Errorf("grpc listen %s: %w", addr, err)
			return
		}
		log.Info("grpc server listening", slog.String("addr", addr))
		if err := grpcSrv.Serve(lis); err != nil {
			grpcErr <- err
		}
	}()

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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", slog.String("error", err.Error()))
	}
	grpcSrv.GracefulStop()
	log.Info("auth service stopped")
}

func buildHTTPServer(
	log *slog.Logger,
	port int,
	h *handlers.AuthHandlers,
	bl middleware.TokenChecker,
	secret string,
) *http.Server {
	mux := http.NewServeMux()
	prefix := api.ApiPrefix

	mux.HandleFunc("POST "+prefix+"/auth/register", h.HandleRegister)
	mux.HandleFunc("POST "+prefix+"/auth/login", h.HandleLogin)
	mux.HandleFunc("POST "+prefix+"/auth/refresh", h.HandleRefresh)
	mux.HandleFunc("GET "+prefix+"/users/{id}", h.HandleGetPublicProfile)

	authMW := middleware.AuthMiddleware(log, bl, secret)
	mux.Handle("POST "+prefix+"/auth/logout", authMW(http.HandlerFunc(h.HandleLogout)))
	mux.Handle("GET "+prefix+"/profile", authMW(http.HandlerFunc(h.HandleGetProfile)))
	mux.Handle("PATCH "+prefix+"/profile", authMW(http.HandlerFunc(h.HandleUpdateProfile)))
	mux.Handle("POST "+prefix+"/profile/avatar", authMW(http.HandlerFunc(h.HandleUploadAvatar)))

	// /metrics — Prometheus scrape endpoint, в обход CSRF и AccessLog (см. middleware/access_log.go).
	mux.Handle("GET /metrics", metrics.Handler())

	httpMetrics := metrics.New("auth")

	// Цепочка middleware (снаружи внутрь): CORS -> RequestID -> Metrics -> AccessLog -> CSRF -> mux
	handler := middleware.CSRFMiddleware(mux)
	handler = middleware.AccessLogMiddleware(log)(handler)
	handler = httpMetrics.Middleware(handler)
	handler = middleware.RequestIDMiddleware(handler)
	handler = middleware.CORSMiddleware(handler)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func buildGRPCServer(
	log *slog.Logger,
	userStorage authgrpc.UserProvider,
	authUC authgrpc.TokenValidator,
) *grpclib.Server {
	grpcMetrics := metrics.NewGRPC("auth")
	srv := grpclib.NewServer(grpclib.UnaryInterceptor(grpcMetrics.UnaryServerInterceptor()))
	authv1.RegisterAuthServiceServer(srv, authgrpc.NewServer(log, userStorage, authUC))
	return srv
}

func splitBrokers(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}
