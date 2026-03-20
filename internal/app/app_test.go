package app

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/clover/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestAppNew_StorageError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	// Invalid DSN for PostgreSQL that should fail initialization
	invalidDSN := "postgres://invalid:user@localhost:5432/nonexistent?sslmode=disable"

	assert.Panics(t, func() {
		New(
			ctx,
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
