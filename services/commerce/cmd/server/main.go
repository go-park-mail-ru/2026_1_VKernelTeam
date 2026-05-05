// Commerce service entry point.
//
// HTTP — корзина, заказы, чаты. gRPC-сервера нет (commerce ничего не отдаёт другим
// сервисам). gRPC-клиенты к Auth (валидация JWT) и Catalog (детали товаров,
// смена статуса). Kafka consumer на clover.catalog.ad-events для cleanup корзин.
package main

import (
	"context"
	"fmt"
	"log/slog"
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

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/config"
	commercegrpc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/handlers"
	commercekafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/kafka"
	commercemw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/delivery/middleware"
	cartrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/cart"
	chatrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/chat"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/repository/postgres"
	cartusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/cart"
	chatusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/commerce/internal/usecase/chat"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logger.SetupLogger(cfg.Env)
	log.Info("commerce service starting", slog.Int("http_port", cfg.HTTP.Port))

	pg, err := postgres.New(cfg.DatabaseDSN)
	if err != nil {
		log.Error("postgres connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pg.Close()

	authClient, err := commercegrpc.NewAuthClient(cfg.AuthGRPCAddr)
	if err != nil {
		log.Error("auth grpc client failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = authClient.Close() }()

	catalogClient, err := commercegrpc.NewCatalogClient(cfg.CatalogGRPCAddr)
	if err != nil {
		log.Error("catalog grpc client failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = catalogClient.Close() }()

	cartStorage := cartrepo.NewCartStorage(pg.Pool, log)
	chatStorage := chatrepo.NewChatStorage(pg.Pool, log)

	// catalogClient реализует AdsProvider/AdProvider (метод GetAdByID).
	cartUC := cartusecase.New(log, cartStorage, catalogClient)
	chatUC := chatusecase.New(log, chatStorage, catalogClient)

	cartHandlers := handlers.NewCartHandlers(log, cartUC)
	chatHandlers := handlers.NewChatHandlers(log, chatUC)

	brokers := splitBrokers(cfg.Kafka.Brokers)
	kafkaConsumer := commercekafka.NewConsumer(brokers, cfg.Kafka.GroupID, cartStorage, log)
	defer func() { _ = kafkaConsumer.Close() }()

	httpSrv := buildHTTPServer(log, cfg.HTTP.Port, cartHandlers, chatHandlers, authClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", httpSrv.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErr <- err
		}
	}()
	go kafkaConsumer.Run(ctx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-stop:
		log.Info("received signal", slog.String("signal", sig.String()))
	case err := <-httpErr:
		log.Error("http server failed", slog.String("error", err.Error()))
	}

	log.Info("shutting down")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", slog.String("error", err.Error()))
	}
	log.Info("commerce service stopped")
}

func buildHTTPServer(
	log *slog.Logger,
	port int,
	cart *handlers.CartHandlers,
	chat *handlers.ChatHandlers,
	authClient *commercegrpc.AuthClient,
) *http.Server {
	mux := http.NewServeMux()
	prefix := api.ApiPrefix

	authMW := commercemw.GRPCAuthMiddleware(log, authClient)

	// Cart
	mux.Handle("GET "+prefix+"/cart", authMW(http.HandlerFunc(cart.HandleGetCart)))
	mux.Handle("POST "+prefix+"/cart", authMW(http.HandlerFunc(cart.HandleAddToCart)))
	mux.Handle("DELETE "+prefix+"/cart/{id}", authMW(http.HandlerFunc(cart.HandleRemoveFromCart)))

	// Chat & orders
	mux.Handle("POST "+prefix+"/ads/{id}/order", authMW(http.HandlerFunc(chat.HandleCreateOrder)))
	mux.Handle("POST "+prefix+"/chats/{id}/confirm", authMW(http.HandlerFunc(chat.HandleConfirmOrder)))
	mux.Handle("GET "+prefix+"/chats", authMW(http.HandlerFunc(chat.HandleGetAllChats)))
	mux.Handle("GET "+prefix+"/chats/{id}", authMW(http.HandlerFunc(chat.HandleGetChat)))

	// /metrics - Prometheus scrape endpoint, в обход CSRF и AccessLog (см. middleware/access_log.go).
	mux.Handle("GET /metrics", metrics.Handler())

	httpMetrics := metrics.New("commerce")

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
