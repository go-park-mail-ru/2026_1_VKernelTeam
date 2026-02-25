package main

import (
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/logger"
)

func main() {
	cfg := config.MustLoadConfig()

	log := logger.SetupLogger(cfg.Env)

	log.Info("starting applications")
	application := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL)
	application.GRPCSrv.MustRun()
	log.Info("applications started")


}
