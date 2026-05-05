// Support service entry point.
//
// Поднимает HTTP-сервер для тикетов техподдержки и Kafka-consumer для
// событий user.deleted из Auth (анонимизация данных удалённых пользователей).
// Авторизация и проверка ролей делегированы Auth-сервису через gRPC.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/api"
	sharedmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/http/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/logger"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/pkg/shared/metrics"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/config"
	authgrpc "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/delivery/handlers"
	supportkafka "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/delivery/kafka"
	supportmw "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/delivery/middleware"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/postgres"
	messagerepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/support_message"
	ticketrepo "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/repository/support_ticket"
	messageusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_message"
	ticketusecase "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/services/support/internal/usecase/support_ticket"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logger.SetupLogger(cfg.Env)
	log.Info("support service starting", slog.Int("http_port", cfg.HTTP.Port))

	pg, err := postgres.New(cfg.DatabaseDSN)
	if err != nil {
		log.Error("postgres connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pg.Close()

	authClient, err := authgrpc.NewAuthClient(cfg.AuthGRPCAddr)
	if err != nil {
		log.Error("auth grpc client failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = authClient.Close() }()

	ticketStorage := ticketrepo.NewSupportTicketStorage(pg.Pool, log)
	messageStorage := messagerepo.NewSupportMessageStorage(pg.Pool, log)

	ticketUC := ticketusecase.New(log, ticketStorage)
	messageUC := messageusecase.New(log, messageStorage, ticketStorage, authClient)

	supportHandlers := handlers.NewSupportTicketHandlers(log, &handlers.Services{
		SupportTicket:  ticketUC,
		SupportMessage: messageUC,
	})

	brokers := splitBrokers(cfg.Kafka.Brokers)
	kafkaConsumer := supportkafka.NewConsumer(brokers, cfg.Kafka.GroupID, pg.Pool, log)
	defer func() { _ = kafkaConsumer.Close() }()

	httpSrv := buildHTTPServer(log, cfg.HTTP.Port, supportHandlers, authClient)

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
	cancel() // останавливает kafka consumer
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", slog.String("error", err.Error()))
	}
	log.Info("support service stopped")
}

func buildHTTPServer(
	log *slog.Logger,
	port int,
	h *handlers.SupportTicketHandlers,
	authClient *authgrpc.AuthClient,
) *http.Server {
	mux := http.NewServeMux()
	prefix := api.ApiPrefix

	authMW := supportmw.GRPCAuthMiddleware(log, authClient)
	staffMW := sharedmw.RoleMiddleware(log, authClient, "support", "admin")

	mux.Handle("POST "+prefix+"/support/tickets", authMW(http.HandlerFunc(h.HandleCreateTicket)))
	mux.Handle("GET "+prefix+"/support/tickets", authMW(http.HandlerFunc(h.HandleGetMyTickets)))
	mux.Handle("GET "+prefix+"/support/tickets/{id}", authMW(http.HandlerFunc(h.HandleGetTicket)))
	mux.Handle("PUT "+prefix+"/support/tickets/{id}", authMW(http.HandlerFunc(h.HandleUpdateTicket)))
	mux.Handle("POST "+prefix+"/support/tickets/{id}/rate", authMW(http.HandlerFunc(h.HandleRateTicket)))
	mux.Handle("POST "+prefix+"/support/tickets/{id}/messages", authMW(http.HandlerFunc(h.HandleSendMessage)))
	mux.Handle("GET "+prefix+"/support/tickets/{id}/messages", authMW(http.HandlerFunc(h.HandleGetMessages)))

	mux.Handle("PATCH "+prefix+"/support/tickets/{id}/status", authMW(staffMW(http.HandlerFunc(h.HandleChangeStatus))))
	mux.Handle("GET "+prefix+"/support/tickets/all", authMW(staffMW(http.HandlerFunc(h.HandleGetAllTickets))))
	mux.Handle("GET "+prefix+"/support/tickets/stats", authMW(staffMW(http.HandlerFunc(h.HandleGetStats))))

	// /metrics — Prometheus scrape endpoint, в обход CSRF и AccessLog (см. middleware/access_log.go).
	mux.Handle("GET /metrics", metrics.Handler())

	httpMetrics := metrics.New("support")

	// CORS -> RequestID -> Metrics -> AccessLog -> CSRF -> mux
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
