package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tempDir := t.TempDir()
	storagePath := filepath.Join(tempDir, "storage.db")

	app := New(
		logger,
		8080,
		storagePath,
		time.Hour,
		time.Minute,
		"test_secret",
	)

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
			8080,
			storagePath,
			time.Hour,
			time.Minute,
			"test_secret",
		)
	})
}
