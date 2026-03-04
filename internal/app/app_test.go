package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_VKernelTeam/sso/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "storage.db")

	cfg := &config.Config{
		StoragePath:     storagePath,
		TokenTTL:        time.Hour,
		CleanupInterval: time.Minute,
		TokenSecret:     "test_secret",
		HTTP: config.HTTPConfig{
			Port: 8080,
		},
	}

	app := New(logger, cfg)

	require.NotNil(t, app)
	require.NotNil(t, app.HTTPServer)
}

func TestAppNew_StorageError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Invalid path for storage that should fail initialization
	storagePath := "/invalid/path/that/does/not/exist/storage.db"

	assert.Panics(t, func() {
		New(
			logger,
			&config.Config{
				StoragePath:     storagePath,
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
