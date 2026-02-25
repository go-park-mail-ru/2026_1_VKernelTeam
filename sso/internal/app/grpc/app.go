package grpcapp

import (
	"fmt"
	"log/slog"
	"net"

	authrpc "github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/grpc/auth"
	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(log *slog.Logger, port int) *App {
	gRPCServer := grpc.NewServer()
	authrpc.RegisterServerApi(gRPCServer)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}


func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With("op", op, "port", a.port)
	log.Info("starting gRPC server")

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC server started")

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.Info("stopping gRPC server")
	a.gRPCServer.GracefulStop()
	a.log.Info("gRPC server stopped")
}
