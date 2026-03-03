// Входная точка приложения. Загружает конфигурацию, настраивает логгер,
// инициализирует приложение и стартует HTTP-сервер.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/logger"
)

func main() {
	cfg := config.MustLoadConfig()

	log := logger.SetupLogger(cfg.Env)

	log.Info("starting applications")
	// convert custom Duration type back to time.Duration
	application := app.New(log, cfg.HTTP.Port, cfg.StoragePath, cfg.TokenTTL.ToDuration(), cfg.CleanupInterval.ToDuration(), cfg.TokenSecret)
	go application.HTTPServer.MustRun()
	log.Info("applications started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop
	log.Info("received signal", "signal", sign)

	log.Info("stopping applications")
	application.HTTPServer.Stop()
	log.Info("applications stopped")
}
