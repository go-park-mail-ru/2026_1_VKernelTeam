package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	grpcapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/app/grpc"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/storage"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, tokenTTL time.Duration) (*App, error) {
	// инициализируем общий слой хранилища; один и тот же объект реализует все
	// интерфейсы, необходимые для auth.
	store, err := storage.New(storagePath)
	if err != nil {
		return nil, err
	}

	// создаем приложение по умолчанию, если его еще нет. в случае, если хранилище
	// уже содержит записи, вызов вернет ErrAppExists, который мы можем
	// безопасно проигнорировать; это упрощает инициализацию и позволяет избежать
	// раскрытия внутренних деталей пакета storage.
	if id, err := store.CreateApp(context.Background(), "default", "secret"); err != nil {
		if !errors.Is(err, storage.ErrAppExists) {
			return nil, err
		}
	} else {
		log.Info("seeded default application", "app_id", id)
	}

	authService := auth.New(log, store, store, store, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}, nil
}
