// Входная точка приложения. Загружает конфигурацию, настраивает логгер,
// инициализирует приложение и стартует HTTP-сервер.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/logger"
)

// @title Clover API
// @version 1.0
// @description API for the Clover service.
// @host clover-go.ru
// @BasePath /api/v1
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name token
func main() {
	cfg := config.MustLoadConfig()

	log := logger.SetupLogger(cfg.Env)

	log.Info("starting applications")
	// convert string to time.Duration

	application := app.New(log, cfg)
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
