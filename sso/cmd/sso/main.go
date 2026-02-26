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
	application, err := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL)
	if err != nil {
		log.Error("failed to initialize application", "err", err)
		os.Exit(1)
	}
	go application.GRPCSrv.MustRun()
	log.Info("applications started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop
	log.Info("received signal", "signal", sign)

	log.Info("stopping applications")
	application.GRPCSrv.Stop()
	log.Info("applications stopped")

}
