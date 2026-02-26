package app

import (
	"log/slog"
	"time"

	grpcapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app/grpc"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, tokenTTL time.Duration) *App {
	// TODO: Initialize storage (UserSaver, UserProvider, AppProvider)
	authService := auth.New(log, nil, nil, nil, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}
