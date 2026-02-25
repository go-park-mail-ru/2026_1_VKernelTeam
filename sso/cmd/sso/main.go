package main

import (
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/logger"
)

func main() {
	cfg := config.MustLoadConfig()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting applications")
}
