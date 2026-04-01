package app

import (
	"io"
	"log/slog"
	"testing"
	"time"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/redis"
	"github.com/stretchr/testify/assert"
)

func TestApp_Stop(t *testing.T) {
	nopLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rc := redis.New("localhost:6379")
	hApp := &httpapp.App{}

	hApp = httpapp.New(
		nopLogger,
		httpapp.Services{},
		nil,
		8080,
		time.Minute,
		time.Minute,
		"secret",
	)

	db := &postgres.Client{}

	a := &App{
		HTTPServer: hApp,
		RedisCache: rc,
		dbClient:   db,
	}

	assert.NotPanics(t, func() {
		a.Stop()
	})
}

func TestAppNew_Panic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	invalidCfg := &config.Config{
		DatabaseDSN: "invalid_dsn",
		RedisAddr:   "localhost:6379",
	}

	assert.Panics(t, func() {
		New(logger, invalidCfg)
	})
}

func TestApp_ManualBuildStop(t *testing.T) {
	a := &App{}

	assert.NotPanics(t, func() {
		a.Stop()
	})
}
