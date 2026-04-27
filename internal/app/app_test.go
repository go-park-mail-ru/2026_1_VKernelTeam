package app

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	httpapp "github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/app/http"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/repository/redis"
	"github.com/stretchr/testify/assert"
)

func TestAppNew_StorageError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Invalid DSN for PostgreSQL that should fail initialization
	invalidDSN := "postgres://invalid:user@localhost:5432/nonexistent?sslmode=disable"

	assert.Panics(t, func() {
		New(
			logger,
			&config.Config{
				DatabaseDSN:     invalidDSN,
				TokenTTL:        time.Hour,
				CleanupInterval: time.Minute,
				TokenSecret:     "test_secret",
				HTTP: config.HTTPConfig{
					Port: 8080,
				},
			},
		)
	})
}

func TestApp_Stop(t *testing.T) {
	nopLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Инициализируем зависимости максимально "безопасно" для тестов
	rc := redis.New("localhost:6379")

	hApp := httpapp.New(
		nopLogger,
		httpapp.Services{},
		nil,
		nil,
		8080,
		time.Minute,
		time.Minute,
		"secret",
	)

	// Чтобы postgres.Client.Close() не паниковал, нам нужно либо иметь
	// инициализированный пул внутри, либо проверку на nil в методе Stop.
	// Если ты добавил проверку на nil в app.go, этот dbClient будет ок:
	db := &postgres.Client{
		Pool: nil, // Специально оставляем nil для проверки безопасности
	}

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
	}

	assert.Panics(t, func() {
		New(logger, invalidCfg)
	})
}

func TestApp_FullInit_CoverageBoost(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := &config.Config{
		TokenTTL:    time.Hour,
		RefreshTTL:  time.Hour,
		TokenSecret: "secret",
		RedisAddr:   "localhost:6379",
		HTTP:        config.HTTPConfig{Port: 8080},
	}

	mockDb := &postgres.Client{
		Pool: nil,
	}

	assert.NotPanics(t, func() {
		a := New(logger, cfg, mockDb)

		assert.NotNil(t, a)
		assert.NotNil(t, a.HTTPServer)
	})
}
